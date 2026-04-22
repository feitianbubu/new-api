package dto

import (
	"strings"

	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type InteractionsRequest struct {
	Input                 any    `json:"input,omitempty"`
	Agent                 string `json:"agent,omitempty"`
	Model                 string `json:"model,omitempty"`
	PreviousInteractionID string `json:"previous_interaction_id,omitempty"`
	Background            *bool  `json:"background,omitempty"`
	Store                 *bool  `json:"store,omitempty"`
	Stream                *bool  `json:"stream,omitempty"`
}

func (r *InteractionsRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{
		TokenType: types.TokenTypeTokenizer,
	}
}

func (r *InteractionsRequest) IsStream(c *gin.Context) bool {
	return r.Stream != nil && *r.Stream
}

func (r *InteractionsRequest) SetModelName(modelName string) {
	if strings.TrimSpace(r.Agent) != "" {
		r.Agent = modelName
		return
	}
	r.Model = modelName
}
