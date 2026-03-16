package jimeng

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

var omniHumanModels = []string{
	"jimeng_realman_avatar_picture_omni_v15",
}

const omniHumanUploadPurpose = "jimeng_omnihuman_file"

func isOmniHumanModel(name string) bool {
	for _, modelName := range omniHumanModels {
		if name == modelName {
			return true
		}
	}
	return false
}

func tryConvertOmniHumanFilesToURLs(c *gin.Context, req *relaycommon.TaskSubmitReq) error {
	if !isOmniHumanModel(req.Model) || c.ContentType() != gin.MIMEMultipartPOSTForm {
		return nil
	}

	for _, fieldName := range []string{"image_url", "audio_url"} {
		currentValue, _ := req.Metadata[fieldName].(string)
		fileURL, converted, err := relaychannel.TryConvertMultipartSingleFileFieldToURL(
			c,
			currentValue,
			[]string{fieldName},
			relaychannel.ImageUploadOptions{
				Purpose:        omniHumanUploadPurpose,
				ExpiresSeconds: 3600,
			},
		)
		if err != nil {
			return err
		}
		if !converted {
			continue
		}
		if req.Metadata == nil {
			req.Metadata = make(map[string]interface{})
		}
		req.Metadata[fieldName] = fileURL
	}
	return nil
}

func resolveOmniHumanBillingSeconds(c *gin.Context, reqModel string, body *requestPayload) (int, bool, error) {
	if !isOmniHumanModel(reqModel) {
		return 0, false, nil
	}
	if strings.TrimSpace(body.ImageURL) == "" {
		return 0, true, fmt.Errorf("image_url is required")
	}
	if strings.TrimSpace(body.AudioURL) == "" {
		return 0, true, fmt.Errorf("audio_url is required")
	}
	seconds, err := resolveOmniHumanAudioDuration(c, body.AudioURL)
	if err != nil {
		return 0, true, err
	}
	return seconds, true, nil
}

func resolveOmniHumanAudioDuration(c *gin.Context, audioURL string) (int, error) {
	source := types.NewURLFileSource(audioURL)
	cachedData, err := service.LoadFileSource(c, source, "omnihuman_audio_duration")
	if err != nil {
		return 0, err
	}

	base64Data, err := cachedData.GetBase64Data()
	if err != nil {
		return 0, err
	}
	audioBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return 0, err
	}

	ext := common.ResolveSupportedAudioExtension(audioURL, cachedData.MimeType)
	if ext == "" {
		return 0, fmt.Errorf("unsupported audio format for duration parsing")
	}

	readSeeker := bytes.NewReader(audioBytes)
	duration, err := common.GetAudioDuration(c, readSeeker, ext)
	if err != nil {
		return 0, err
	}
	return int(math.Ceil(duration)), nil
}
