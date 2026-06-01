package tos

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/storage/common"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"resty.dev/v3"
)

const DefaultStorageSeconds = 24 * 60 * 60
const maxUploadSizeBytes = 100 * 1024 * 1024

func parseExpirationFromHeader(headerValue string) (time.Time, error) {
	if headerValue == "" {
		return time.Time{}, fmt.Errorf("empty expiration header")
	}

	start := strings.Index(headerValue, `"`)
	if start == -1 {
		return time.Time{}, fmt.Errorf("invalid expiration header format: missing opening quote")
	}
	start++

	end := strings.Index(headerValue[start:], `"`)
	if end == -1 {
		return time.Time{}, fmt.Errorf("invalid expiration header format: missing closing quote")
	}

	dateStr := headerValue[start : start+end]
	expirationTime, err := time.Parse(time.RFC1123, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse expiration date: %w", err)
	}

	return expirationTime, nil
}

func getExpirationFromTOSResponse(headResult *tos.HeadObjectV2Output) (time.Time, bool) {
	if headResult != nil && headResult.ObjectMetaV2.Expiration != "" {
		if expTime, err := parseExpirationFromHeader(headResult.ObjectMetaV2.Expiration); err == nil {
			return expTime, true
		}
	}
	return time.Time{}, false
}

func extractObjectKeyFromFileID(fileID string) (string, error) {
	if !strings.HasPrefix(fileID, "file-") {
		return "", fmt.Errorf("invalid file_id format")
	}

	// Decode base64 encoded objectKey
	encodedKey := fileID[5:]
	decodedBytes, err := base64.URLEncoding.DecodeString(encodedKey)
	if err != nil {
		return "", fmt.Errorf("invalid file_id encoding: %w", err)
	}

	return string(decodedBytes), nil
}

func generateFileID(objectKey string) string {
	// Encode objectKey as base64 to ensure URL-safe file ID
	encodedKey := base64.URLEncoding.EncodeToString([]byte(objectKey))
	return fmt.Sprintf("file-%s", encodedKey)
}

func extractMetadata(meta tos.Metadata) (purpose, filename string, metadata map[string]string) {
	metadata = make(map[string]string)

	if meta != nil {
		purpose, _ = meta.Get("purpose")
		filename, _ = meta.Get("original-filename")

		meta.Range(func(key, value string) bool {
			metadata[key] = value
			return true
		})
	}
	return
}

func calculateExpiresAt(headResult *tos.HeadObjectV2Output) time.Time {
	if expTime, found := getExpirationFromTOSResponse(headResult); found {
		return expTime
	}
	return time.Time{}
}

func parseSizeFromContentRange(contentRange string) (int64, error) {
	if contentRange == "" {
		return 0, fmt.Errorf("content-range missing")
	}
	slash := strings.LastIndex(contentRange, "/")
	if slash == -1 || slash == len(contentRange)-1 {
		return 0, fmt.Errorf("invalid content-range format")
	}
	sizeStr := strings.TrimSpace(contentRange[slash+1:])
	if sizeStr == "*" {
		return 0, fmt.Errorf("content-range size is unknown")
	}
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil || size < 0 {
		return 0, fmt.Errorf("invalid content-range size")
	}
	return size, nil
}

func getSizeByHead(ctx context.Context, sourceURL string) (int64, error) {
	resp, err := resty.New().R().SetContext(ctx).Head(sourceURL)
	if err != nil {
		return 0, fmt.Errorf("head request failed: %w", err)
	}
	if resp.StatusCode() >= http.StatusBadRequest {
		return 0, fmt.Errorf("head request returned status %d", resp.StatusCode())
	}
	clHeader := resp.Header().Get("Content-Length")
	if clHeader == "" {
		return 0, fmt.Errorf("content-length missing in head response")
	}
	size, err := strconv.ParseInt(clHeader, 10, 64)
	if err != nil || size < 0 {
		return 0, fmt.Errorf("invalid content-length in head response")
	}
	return size, nil
}

func getSizeByRangeGet(ctx context.Context, sourceURL string) (int64, error) {
	resp, err := resty.New().R().SetContext(ctx).SetHeader("Range", "bytes=0-0").Get(sourceURL)
	if err != nil {
		return 0, fmt.Errorf("range get request failed: %w", err)
	}
	if resp.StatusCode() >= http.StatusBadRequest {
		return 0, fmt.Errorf("range get request returned status %d", resp.StatusCode())
	}
	if contentRange := resp.Header().Get("Content-Range"); contentRange != "" {
		return parseSizeFromContentRange(contentRange)
	}
	clHeader := resp.Header().Get("Content-Length")
	if clHeader == "" {
		return 0, fmt.Errorf("content-length missing in range get response")
	}
	size, err := strconv.ParseInt(clHeader, 10, 64)
	if err != nil || size < 0 {
		return 0, fmt.Errorf("invalid content-length in range get response")
	}
	return size, nil
}

type FileObject = common.FileObject
type FileListResult = common.FileListResult
type UploadOptions = common.UploadOptions
type ListOptions = common.ListOptions
type FileContent = common.FileContent
type ExpiresAfter = common.ExpiresAfter

type TOSStorage struct {
	client *tos.ClientV2
	config Config
}

func NewTOSStorage(config Config) (*TOSStorage, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	client, err := tos.NewClientV2(
		config.Endpoint,
		tos.WithRegion(config.Region),
		tos.WithCredentials(tos.NewStaticCredentials(config.AccessKey, config.SecretKey)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create TOS client: %w", err)
	}

	storage := &TOSStorage{
		client: client,
		config: config,
	}

	if err := storage.ensureBucketExists(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return storage, nil
}

func (t *TOSStorage) ensureBucketExists(ctx context.Context) error {
	_, err := t.client.HeadBucket(ctx, &tos.HeadBucketInput{Bucket: t.config.Bucket})
	if err != nil {
		_, err = t.client.CreateBucket(ctx, &tos.CreateBucketInput{Bucket: t.config.Bucket})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	return nil
}

func (t *TOSStorage) UploadFile(ctx context.Context, reader io.Reader, size int64, opts UploadOptions) (*FileObject, error) {
	if size > maxUploadSizeBytes {
		return nil, fmt.Errorf("file size exceeds maximum limit of 100MB")
	}

	now := time.Now()

	// 支持自定义 objectKey：如果 opts.ObjectKey 不为空则使用，否则使用默认生成逻辑
	var objectKey string
	if opts.ObjectKey != "" {
		objectKey = opts.ObjectKey
	} else {
		objectKey = fmt.Sprintf("uploads/%d/%d%s", opts.UserID, now.UnixMilli(), filepath.Ext(opts.Filename))
	}

	metadata := buildUploadMetadata(opts, now)

	contentType := opts.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	putInput := &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket:        t.config.Bucket,
			Key:           objectKey,
			ContentType:   contentType,
			ContentLength: size,
			Meta:          metadata,
		},
		Content: reader,
	}

	if opts.ExpiresAfter != nil {
		objectExpiresDays := int64(math.Ceil(float64(opts.ExpiresAfter.Seconds) / 86400.0))
		putInput.ObjectExpires = objectExpiresDays
	}

	result, err := t.client.PutObjectV2(ctx, putInput)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file to TOS: %w", err)
	}

	fileObj := &FileObject{
		ID:          generateFileID(objectKey),
		Object:      "file",
		Bytes:       size,
		CreatedAt:   now,
		Filename:    opts.Filename,
		Purpose:     opts.Purpose,
		ETag:        result.ETag,
		Key:         objectKey,
		ContentType: contentType,
		Metadata:    metadata,
	}

	// 只有在设置了过期时间时才计算 ExpiresAt
	if opts.ExpiresAfter != nil {
		objectExpiresDays := int64(math.Ceil(float64(opts.ExpiresAfter.Seconds) / 86400.0))
		fileObj.ExpiresAt = now.Truncate(24*time.Hour).AddDate(0, 0, int(objectExpiresDays)+1)
	}

	return fileObj, nil
}

func (t *TOSStorage) UploadFileByURL(ctx context.Context, sourceURL string, opts UploadOptions) (*FileObject, error) {
	if sourceURL == "" {
		return nil, fmt.Errorf("source url is required")
	}

	now := time.Now()

	size, err := getSizeByHead(ctx, sourceURL)
	if err != nil {
		if fallbackSize, fallbackErr := getSizeByRangeGet(ctx, sourceURL); fallbackErr == nil {
			size = fallbackSize
		} else {
			return nil, fmt.Errorf("head failed: %v; range get failed: %v", err, fallbackErr)
		}
	}
	if size > maxUploadSizeBytes {
		return nil, fmt.Errorf("file size exceeds maximum limit of 100MB")
	}

	filename := opts.Filename
	if filename == "" {
		if parsedURL, err := url.Parse(sourceURL); err == nil && parsedURL != nil {
			base := filepath.Base(parsedURL.Path)
			if base != "." && base != "/" {
				filename = base
			}
		}
		if filename == "" {
			filename = "file"
		}
	}
	opts.Filename = filename

	var objectKey string
	if opts.ObjectKey != "" {
		objectKey = opts.ObjectKey
	} else {
		objectKey = fmt.Sprintf("uploads/%d/%d%s", opts.UserID, now.UnixMilli(), filepath.Ext(filename))
	}

	metadata := buildUploadMetadata(opts, now)

	fetchInput := &tos.FetchObjectInputV2{
		Bucket: t.config.Bucket,
		Key:    objectKey,
		URL:    sourceURL,
		Meta:   metadata,
	}

	if opts.ExpiresAfter != nil {
		objectExpiresDays := int64(math.Ceil(float64(opts.ExpiresAfter.Seconds) / 86400.0))
		fetchInput.GenericInput.RequestHeader = map[string]string{
			"X-Tos-Object-Expires": strconv.FormatInt(objectExpiresDays, 10),
		}
	}

	result, err := t.client.FetchObjectV2(ctx, fetchInput)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch object to TOS: %w", err)
	}

	headResult, err := t.client.HeadObjectV2(ctx, &tos.HeadObjectV2Input{
		Bucket: t.config.Bucket,
		Key:    objectKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	purpose, extractedFilename, metadata := extractMetadata(headResult.Meta)
	if extractedFilename == "" {
		extractedFilename = filename
	}
	if purpose == "" {
		purpose = opts.Purpose
	}

	contentType := headResult.ContentType
	if contentType == "" {
		contentType = opts.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	fileObj := &FileObject{
		ID:          generateFileID(objectKey),
		Object:      "file",
		Bytes:       headResult.ContentLength,
		CreatedAt:   headResult.LastModified,
		Filename:    extractedFilename,
		Purpose:     purpose,
		ETag:        result.Etag,
		Key:         objectKey,
		ContentType: contentType,
		Metadata:    metadata,
	}

	if opts.ExpiresAfter != nil {
		objectExpiresDays := int64(math.Ceil(float64(opts.ExpiresAfter.Seconds) / 86400.0))
		fileObj.ExpiresAt = now.Truncate(24*time.Hour).AddDate(0, 0, int(objectExpiresDays)+1)
	}

	return fileObj, nil
}

func (t *TOSStorage) ListFiles(ctx context.Context, opts ListOptions) (*FileListResult, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	listInput := &tos.ListObjectsV2Input{
		Bucket: t.config.Bucket,
		ListObjectsInput: tos.ListObjectsInput{
			Prefix:  lo.Ternary(opts.Prefix != "", opts.Prefix, fmt.Sprintf("uploads/%d/", opts.UserID)),
			MaxKeys: limit,
			Marker:  opts.After,
		},
	}

	result, err := t.client.ListObjectsV2(ctx, listInput)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects from TOS: %w", err)
	}

	var files []FileObject
	for _, obj := range result.Contents {
		headResult, err := t.client.HeadObjectV2(ctx, &tos.HeadObjectV2Input{
			Bucket: t.config.Bucket,
			Key:    obj.Key,
		})
		if err != nil {
			continue
		}

		purpose, filename, metadata := extractMetadata(headResult.Meta)

		// 只有在明确指定了 Purpose 时才过滤
		if opts.Purpose != "" && purpose != opts.Purpose {
			continue
		}

		if filename == "" {
			filename = filepath.Base(obj.Key)
		}

		files = append(files, FileObject{
			ID:          generateFileID(obj.Key),
			Object:      "file",
			Bytes:       obj.Size,
			CreatedAt:   obj.LastModified,
			ExpiresAt:   calculateExpiresAt(headResult),
			Filename:    filename,
			ContentType: headResult.ContentType,
			Purpose:     purpose,
			ETag:        obj.ETag,
			Key:         obj.Key,
			Metadata:    metadata,
		})
	}

	return &FileListResult{
		Files:      files,
		HasMore:    result.IsTruncated,
		NextMarker: result.NextMarker,
	}, nil
}

func (t *TOSStorage) GetFileInfo(ctx context.Context, fileID string) (*FileObject, error) {
	objectKey, err := extractObjectKeyFromFileID(fileID)
	if err != nil {
		return nil, err
	}

	headResult, err := t.client.HeadObjectV2(ctx, &tos.HeadObjectV2Input{
		Bucket: t.config.Bucket,
		Key:    objectKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	purpose, filename, metadata := extractMetadata(headResult.Meta)

	if filename == "" {
		filename = filepath.Base(objectKey)
	}

	return &FileObject{
		ID:        fileID,
		Object:    "file",
		Bytes:     headResult.ContentLength,
		CreatedAt: headResult.LastModified,
		ExpiresAt: calculateExpiresAt(headResult),
		Filename:  filename,
		Purpose:   purpose,
		ETag:      headResult.ETag,
		Key:       objectKey,
		Metadata:  metadata,
	}, nil
}

func (t *TOSStorage) DeleteFile(ctx context.Context, fileID string) error {
	objectKey, err := extractObjectKeyFromFileID(fileID)
	if err != nil {
		return err
	}

	_, err = t.client.DeleteObjectV2(ctx, &tos.DeleteObjectV2Input{
		Bucket: t.config.Bucket,
		Key:    objectKey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from TOS: %w", err)
	}

	return nil
}

func (t *TOSStorage) GetFileContent(ctx context.Context, fileID string) (*FileContent, error) {
	objectKey, err := extractObjectKeyFromFileID(fileID)
	if err != nil {
		return nil, err
	}

	getInput := &tos.GetObjectV2Input{
		Bucket: t.config.Bucket,
		Key:    objectKey,
	}

	var rangeHeader string
	if ginCtx, ok := ctx.(*gin.Context); ok && ginCtx != nil {
		rangeHeader = ginCtx.GetHeader("Range")
		if rangeHeader != "" {
			getInput.Range = rangeHeader
		}
	}

	getResult, err := t.client.GetObjectV2(ctx, getInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get object content from TOS: %w", err)
	}

	headResult, err := t.client.HeadObjectV2(ctx, &tos.HeadObjectV2Input{
		Bucket: t.config.Bucket,
		Key:    objectKey,
	})
	if err != nil {
		getResult.Content.Close()
		return nil, fmt.Errorf("failed to get object metadata: %w", err)
	}

	_, filename, metadata := extractMetadata(headResult.Meta)

	var (
		start int64
		end   int64
		total = headResult.ContentLength
	)
	if cr := getResult.ContentRange; cr != "" {
		if s, e, t, ok := parseContentRange(cr); ok {
			start, end, total = s, e, t
		}
	}

	return &FileContent{
		Content:       getResult.Content,
		ContentType:   getResult.ContentType,
		ContentLength: getResult.ContentLength, // length of returned chunk (or full size if no range)
		TotalLength:   total,
		RangeStart:    start,
		RangeEnd:      end,
		Filename:      filename,
		Metadata:      metadata,
	}, nil
}

func (t *TOSStorage) Close() error {
	return nil
}

func (t *TOSStorage) PresignURL(_ context.Context, objectKey string, expireSeconds int64) (string, error) {
	if expireSeconds <= 0 {
		expireSeconds = int64(DefaultStorageSeconds)
	}
	out, err := t.client.PreSignedURL(&tos.PreSignedURLInput{
		Bucket:     t.config.Bucket,
		Key:        objectKey,
		HTTPMethod: http.MethodGet,
		Expires:    expireSeconds,
	})
	if err != nil {
		return "", fmt.Errorf("presign url failed: %w", err)
	}
	return out.SignedUrl, nil
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

	startStr := strings.TrimSpace(rangePart[:dash])
	endStr := strings.TrimSpace(rangePart[dash+1:])
	totalStr := strings.TrimSpace(totalPart)

	start, err1 := strconv.ParseInt(startStr, 10, 64)
	end, err2 := strconv.ParseInt(endStr, 10, 64)
	total, err3 := strconv.ParseInt(totalStr, 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || start < 0 || end < start || total <= 0 {
		return 0, 0, 0, false
	}
	return start, end, total, true
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
