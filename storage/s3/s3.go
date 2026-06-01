package s3

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/storage/common"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

const DefaultStorageSeconds = 24 * 60 * 60
const maxUploadSizeBytes = 100 * 1024 * 1024

type FileObject = common.FileObject
type FileListResult = common.FileListResult
type UploadOptions = common.UploadOptions
type ListOptions = common.ListOptions
type FileContent = common.FileContent
type ExpiresAfter = common.ExpiresAfter

type S3Storage struct {
	client *s3.Client
	config Config
}

func NewS3Storage(config Config) (*S3Storage, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	config.Endpoint = normalizeEndpoint(config.Endpoint)

	awsCfg := aws.Config{
		Region:      config.Region,
		Credentials: credentials.NewStaticCredentialsProvider(config.AccessKey, config.SecretKey, ""),
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if config.Endpoint != "" {
			o.BaseEndpoint = aws.String(config.Endpoint)
		}
		o.UsePathStyle = config.UsePathStyle
		// Disable the SDK's default CRC32 request checksum / response validation.
		// Many S3-compatible providers (R2, B2, MinIO, Filebase) reject the
		// aws-chunked checksum trailer that "when_supported" would add.
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})

	// Note: bucket is assumed to exist. Unlike the TOS backend we do not
	// auto-create it — most S3-compatible providers reject CreateBucket from
	// scoped credentials or require provider-specific region handling.
	return &S3Storage{client: client, config: config}, nil
}

func (s *S3Storage) UploadFile(ctx context.Context, reader io.Reader, size int64, opts UploadOptions) (*FileObject, error) {
	if size > maxUploadSizeBytes {
		return nil, fmt.Errorf("file size exceeds maximum limit of 100MB")
	}

	now := time.Now()

	objectKey := opts.ObjectKey
	if objectKey == "" {
		objectKey = fmt.Sprintf("uploads/%d/%d%s", opts.UserID, now.UnixMilli(), filepath.Ext(opts.Filename))
	}

	contentType := opts.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	metadata := buildUploadMetadata(opts, now)

	// S3 has no per-object TTL header. We record the requested expiry as the
	// "expires-at" metadata and expose ExpiresAt for client/billing parity with
	// the TOS backend, but physical deletion is NOT enforced — configure a bucket
	// lifecycle rule (keyed on this metadata or the object prefix) to reclaim space.
	var expiresAt time.Time
	if opts.ExpiresAfter != nil {
		days := int64(math.Ceil(float64(opts.ExpiresAfter.Seconds) / 86400.0))
		expiresAt = now.Truncate(24*time.Hour).AddDate(0, 0, int(days)+1)
		metadata["expires-at"] = expiresAt.Format(time.RFC3339)
	}

	putOut, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.config.Bucket),
		Key:           aws.String(objectKey),
		Body:          reader,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
		Metadata:      metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return &FileObject{
		ID:          generateFileID(objectKey),
		Object:      "file",
		Bytes:       size,
		CreatedAt:   now,
		ExpiresAt:   expiresAt,
		Filename:    opts.Filename,
		Purpose:     opts.Purpose,
		ETag:        aws.ToString(putOut.ETag),
		Key:         objectKey,
		ContentType: contentType,
		Metadata:    metadata,
	}, nil
}

func (s *S3Storage) ListFiles(ctx context.Context, opts ListOptions) (*FileListResult, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	prefix := opts.Prefix
	if prefix == "" {
		prefix = fmt.Sprintf("uploads/%d/", opts.UserID)
	}

	listInput := &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.config.Bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(int32(limit)),
	}
	if opts.After != "" {
		listInput.ContinuationToken = aws.String(opts.After)
	}

	result, err := s.client.ListObjectsV2(ctx, listInput)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects from S3: %w", err)
	}

	var files []FileObject
	for _, obj := range result.Contents {
		key := aws.ToString(obj.Key)

		headResult, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(s.config.Bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			continue
		}

		purpose, filename, metadata := extractMetadata(headResult.Metadata)

		if opts.Purpose != "" && purpose != opts.Purpose {
			continue
		}

		if filename == "" {
			filename = filepath.Base(key)
		}

		files = append(files, FileObject{
			ID:          generateFileID(key),
			Object:      "file",
			Bytes:       aws.ToInt64(obj.Size),
			CreatedAt:   aws.ToTime(obj.LastModified),
			ExpiresAt:   expiresAtFromMetadata(metadata),
			Filename:    filename,
			ContentType: aws.ToString(headResult.ContentType),
			Purpose:     purpose,
			ETag:        aws.ToString(obj.ETag),
			Key:         key,
			Metadata:    metadata,
		})
	}

	return &FileListResult{
		Files:      files,
		HasMore:    aws.ToBool(result.IsTruncated),
		NextMarker: aws.ToString(result.NextContinuationToken),
	}, nil
}

func (s *S3Storage) GetFileInfo(ctx context.Context, fileID string) (*FileObject, error) {
	objectKey, err := extractObjectKeyFromFileID(fileID)
	if err != nil {
		return nil, err
	}

	headResult, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	purpose, filename, metadata := extractMetadata(headResult.Metadata)
	if filename == "" {
		filename = filepath.Base(objectKey)
	}

	return &FileObject{
		ID:          fileID,
		Object:      "file",
		Bytes:       aws.ToInt64(headResult.ContentLength),
		CreatedAt:   aws.ToTime(headResult.LastModified),
		ExpiresAt:   expiresAtFromMetadata(metadata),
		Filename:    filename,
		Purpose:     purpose,
		ETag:        aws.ToString(headResult.ETag),
		Key:         objectKey,
		ContentType: aws.ToString(headResult.ContentType),
		Metadata:    metadata,
	}, nil
}

func (s *S3Storage) DeleteFile(ctx context.Context, fileID string) error {
	objectKey, err := extractObjectKeyFromFileID(fileID)
	if err != nil {
		return err
	}

	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %w", err)
	}

	return nil
}

func (s *S3Storage) GetFileContent(ctx context.Context, fileID string) (*FileContent, error) {
	objectKey, err := extractObjectKeyFromFileID(fileID)
	if err != nil {
		return nil, err
	}

	getInput := &s3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(objectKey),
	}

	if ginCtx, ok := ctx.(*gin.Context); ok && ginCtx != nil {
		if rangeHeader := ginCtx.GetHeader("Range"); rangeHeader != "" {
			getInput.Range = aws.String(rangeHeader)
		}
	}

	getResult, err := s.client.GetObject(ctx, getInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get object content from S3: %w", err)
	}

	_, filename, metadata := extractMetadata(getResult.Metadata)

	// GetObject already returns size/range info, so no extra HeadObject is needed.
	total := aws.ToInt64(getResult.ContentLength)
	var start, end int64
	if cr := aws.ToString(getResult.ContentRange); cr != "" {
		if s, e, t, ok := parseContentRange(cr); ok {
			start, end, total = s, e, t
		}
	}

	return &FileContent{
		Content:       getResult.Body,
		ContentType:   aws.ToString(getResult.ContentType),
		ContentLength: aws.ToInt64(getResult.ContentLength),
		TotalLength:   total,
		RangeStart:    start,
		RangeEnd:      end,
		Filename:      filename,
		Metadata:      metadata,
	}, nil
}

func (s *S3Storage) Close() error {
	return nil
}

func (s *S3Storage) PresignURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error) {
	if expireSeconds <= 0 {
		expireSeconds = DefaultStorageSeconds
	}
	presignClient := s3.NewPresignClient(s.client)
	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(objectKey),
	}, s3.WithPresignExpires(time.Duration(expireSeconds)*time.Second))
	if err != nil {
		return "", fmt.Errorf("presign url failed: %w", err)
	}
	return req.URL, nil
}

func extractObjectKeyFromFileID(fileID string) (string, error) {
	if !strings.HasPrefix(fileID, "file-") {
		return "", fmt.Errorf("invalid file_id format")
	}
	decodedBytes, err := base64.URLEncoding.DecodeString(fileID[5:])
	if err != nil {
		return "", fmt.Errorf("invalid file_id encoding: %w", err)
	}
	return string(decodedBytes), nil
}

func generateFileID(objectKey string) string {
	return fmt.Sprintf("file-%s", base64.URLEncoding.EncodeToString([]byte(objectKey)))
}

func extractMetadata(meta map[string]string) (purpose, filename string, metadata map[string]string) {
	metadata = make(map[string]string)
	for k, v := range meta {
		metadata[k] = v
	}
	return metadata["purpose"], metadata["original-filename"], metadata
}

func expiresAtFromMetadata(metadata map[string]string) time.Time {
	if v := metadata["expires-at"]; v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
	}
	return time.Time{}
}

func buildUploadMetadata(opts UploadOptions, now time.Time) map[string]string {
	metadata := map[string]string{
		"original-filename": opts.Filename,
		"upload-time":       now.Format(time.RFC3339),
		"user-id":           strconv.Itoa(opts.UserID),
		"purpose":           opts.Purpose,
	}

	if opts.ExpiresAfter != nil {
		metadata["expires-after-anchor"] = opts.ExpiresAfter.Anchor
		metadata["expires-after-seconds"] = strconv.Itoa(opts.ExpiresAfter.Seconds)
	}

	for k, v := range opts.Metadata {
		metadata[k] = v
	}

	return metadata
}

func parseContentRange(cr string) (int64, int64, int64, bool) {
	const prefix = "bytes "
	if !strings.HasPrefix(cr, prefix) {
		return 0, 0, 0, false
	}
	body := strings.TrimPrefix(cr, prefix)
	slash := strings.Index(body, "/")
	if slash < 0 {
		return 0, 0, 0, false
	}
	rangePart := body[:slash]
	totalPart := body[slash+1:]

	dash := strings.Index(rangePart, "-")
	if dash < 0 {
		return 0, 0, 0, false
	}

	start, err1 := strconv.ParseInt(strings.TrimSpace(rangePart[:dash]), 10, 64)
	end, err2 := strconv.ParseInt(strings.TrimSpace(rangePart[dash+1:]), 10, 64)
	total, err3 := strconv.ParseInt(strings.TrimSpace(totalPart), 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || start < 0 || end < start || total <= 0 {
		return 0, 0, 0, false
	}
	return start, end, total, true
}
