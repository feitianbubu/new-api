package volcengine

import (
	"encoding/json"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/channel"

	"github.com/gin-gonic/gin"
)

func convertImageFileToURL(c *gin.Context, request dto.ImageRequest) (dto.ImageRequest, error) {
	imageFiles, err := channel.ExtractImageFilesFromMultipart(c, []string{"image"})
	if err != nil {
		return request, err
	}

	userID := channel.GetUserIDFromContext(c)

	modifiedRequest, err := common.DeepCopy(&request)
	if err != nil {
		return request, fmt.Errorf("failed to copy image request: %w", err)
	}

	var imageURLs []string
	for _, imageFile := range imageFiles {
		imageURL, err := channel.UploadImageFile(c, imageFile, userID, channel.ImageUploadOptions{
			Purpose:        "volcengine_image",
			ExpiresSeconds: 3600, // 1 hour
		})
		if err != nil {
			return request, fmt.Errorf("failed to upload image file for volcengine: %w", err)
		}
		imageURLs = append(imageURLs, imageURL)
	}

	if len(imageURLs) == 1 {
		modifiedRequest.Image, _ = json.Marshal(imageURLs[0])
	} else {
		modifiedRequest.Image, _ = json.Marshal(imageURLs)
	}

	logger.LogInfo(c, fmt.Sprintf("Converted %d image file(s) to URL(s) for volcengine", len(imageURLs)))

	return *modifiedRequest, nil
}
