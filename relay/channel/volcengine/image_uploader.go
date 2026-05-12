package volcengine

import (
	"encoding/json"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"

	"github.com/gin-gonic/gin"
)

func convertImageFileToURL(c *gin.Context, request dto.ImageRequest) (dto.ImageRequest, error) {
	imageURLs := c.PostFormArray("image")
	if len(imageURLs) == 0 {
		return request, nil
	}

	modifiedRequest, err := common.DeepCopy(&request)
	if err != nil {
		return request, fmt.Errorf("failed to copy image request: %w", err)
	}

	if len(imageURLs) == 1 {
		modifiedRequest.Image, _ = json.Marshal(imageURLs[0])
	} else {
		modifiedRequest.Image, _ = json.Marshal(imageURLs)
	}

	logger.LogInfo(c, fmt.Sprintf("Converted %d image URL(s) for volcengine", len(imageURLs)))

	return *modifiedRequest, nil
}
