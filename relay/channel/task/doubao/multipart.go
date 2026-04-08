package doubao

import (
	relaychannel "github.com/QuantumNous/new-api/relay/channel"
	"github.com/gin-gonic/gin"
)

const videoUploadPurpose = "doubao_video_file"

type mediaFieldDef struct {
	fieldName string
	itemType  string
	role      string
	setURL    func(ci *ContentItem, url string)
}

var mediaFields = []mediaFieldDef{
	{"image_url", "image_url", "reference_image", func(ci *ContentItem, url string) { ci.ImageURL = &MediaURL{URL: url} }},
	{"video_url", "video_url", "reference_video", func(ci *ContentItem, url string) { ci.VideoURL = &MediaURL{URL: url} }},
	{"audio_url", "audio_url", "reference_audio", func(ci *ContentItem, url string) { ci.AudioURL = &MediaURL{URL: url} }},
}

func appendMultipartMediaToContent(c *gin.Context, body *requestPayload) error {
	uploadOpts := relaychannel.ImageUploadOptions{
		Purpose:        videoUploadPurpose,
		ExpiresSeconds: 7200,
	}

	for _, mf := range mediaFields {
		urls, err := relaychannel.TryConvertMultipartFieldsToURLs(c, []string{mf.fieldName}, uploadOpts)
		if err != nil {
			return err
		}

		for _, u := range urls {
			ci := ContentItem{Type: mf.itemType, Role: mf.role}
			mf.setURL(&ci, u)
			body.Content = append(body.Content, ci)
		}
	}

	return nil
}
