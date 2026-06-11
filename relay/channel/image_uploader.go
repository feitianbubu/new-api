package channel

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/QuantumNous/new-api/storage"

	"github.com/gin-gonic/gin"
)

// ImageUploadOptions defines options for uploading an image
type ImageUploadOptions struct {
	Purpose        string // Purpose of the upload (e.g., "volcengine_image", "ali_image")
	ExpiresSeconds int    // Expiration time in seconds
}

func UploadMultipartFile(c *gin.Context, file *multipart.FileHeader, userID int, options ImageUploadOptions) (string, error) {
	fileReader, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer fileReader.Close()

	storageInstance, err := storage.NewStorageFromEnv()
	if err != nil {
		return "", fmt.Errorf("failed to create storage instance: %w", err)
	}

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(file.Filename)))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	var expiresAfter *storage.ExpiresAfter
	if options.ExpiresSeconds > 0 {
		expiresAfter = &storage.ExpiresAfter{
			Anchor:  "created_at",
			Seconds: options.ExpiresSeconds,
		}
	}

	objectKey := fmt.Sprintf("%s/%s", common.GetOSSFilePath(c), file.Filename)
	fileObj, err := storageInstance.UploadFile(c, fileReader, file.Size, storage.UploadOptions{
		Filename:     file.Filename,
		ContentType:  contentType,
		Purpose:      options.Purpose,
		UserID:       userID,
		ObjectKey:    objectKey,
		ExpiresAfter: expiresAfter,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to storage: %w", err)
	}

	fileURL := fmt.Sprintf("%s/v1/files/%s/content", system_setting.ServerAddress, fileObj.ID)

	logger.LogInfo(c, fmt.Sprintf("Successfully uploaded file %s (purpose: %s), URL: %s", fileObj.ID, options.Purpose, fileURL))

	return fileURL, nil
}

func UploadImageFile(c *gin.Context, file *multipart.FileHeader, userID int, options ImageUploadOptions) (string, error) {
	if file.Header.Get("Content-Type") == "" {
		file.Header.Set("Content-Type", DetectImageMimeType(file.Filename))
	}
	return UploadMultipartFile(c, file, userID, options)
}

func UploadBase64Image(c *gin.Context, base64Data string, filename string, userID int, options ImageUploadOptions) (string, error) {
	// 解码base64数据
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image data: %w", err)
	}

	storageInstance, err := storage.NewStorageFromEnv()
	if err != nil {
		return "", fmt.Errorf("failed to create storage instance: %w", err)
	}

	contentType := DetectImageMimeType(filename)

	if options.ExpiresSeconds <= 0 {
		options.ExpiresSeconds = 3600
	}
	expiresAfter := &storage.ExpiresAfter{
		Anchor:  "created_at",
		Seconds: options.ExpiresSeconds,
	}
	// 直接上传文件内容，不需要创建multipart
	objectKey := fmt.Sprintf("%s/%s", common.GetOSSFilePath(c), filename)
	uploadOptions := storage.UploadOptions{
		Filename:     filename,
		ContentType:  contentType,
		Purpose:      options.Purpose,
		UserID:       userID,
		ObjectKey:    objectKey,
		ExpiresAfter: expiresAfter,
	}

	fileObj, err := storageInstance.UploadFile(c, bytes.NewReader(imageData), int64(len(imageData)), uploadOptions)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to storage: %w", err)
	}

	fileURL := fmt.Sprintf("%s/v1/files/%s/content", system_setting.ServerAddress, fileObj.ID)

	logger.LogInfo(c, fmt.Sprintf("Successfully uploaded base64 image file %s (purpose: %s), URL: %s", fileObj.ID, options.Purpose, fileURL))

	return fileURL, nil
}

// GetUserIDFromContext extracts user ID from gin context, with fallback logic
func GetUserIDFromContext(c *gin.Context) int {
	userID := c.GetInt("id")
	if userID == 0 {
		if tokenID := c.GetInt("token_id"); tokenID != 0 {
			userID = 1
		}
	}
	return userID
}

// DetectImageMimeType detects the MIME type of an image file based on its extension
func DetectImageMimeType(filename string) string {
	lowerFilename := strings.ToLower(filename)

	switch {
	case strings.HasSuffix(lowerFilename, ".jpg"), strings.HasSuffix(lowerFilename, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lowerFilename, ".png"):
		return "image/png"
	case strings.HasSuffix(lowerFilename, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lowerFilename, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lowerFilename, ".bmp"):
		return "image/bmp"
	case strings.HasSuffix(lowerFilename, ".svg"):
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

func TryConvertBase64ImagesToURLs(c *gin.Context, openAIResponse *dto.ImageResponse) {
	//imageReq, ok := info.Request.(*dto.ImageRequest)
	//if !ok {
	//	return
	//}
	//if imageReq.ResponseFormat != "url" {
	//	return
	//}
	userID := GetUserIDFromContext(c)
	for i, data := range openAIResponse.Data {
		if data.B64Json != "" && data.Url == "" {
			imageURL, err := UploadBase64Image(c, data.B64Json, "generated_image.png", userID, ImageUploadOptions{
				Purpose: "gemini_generated_image",
			})
			if err != nil {
				logger.LogWarn(c, fmt.Sprintf("Failed to upload base64 image: %v", err))
				continue
			}
			openAIResponse.Data[i].Url = imageURL
			openAIResponse.Data[i].B64Json = ""
		}
	}
	return
}
