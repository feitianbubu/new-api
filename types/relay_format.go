package types

type RelayFormat string

const (
	RelayFormatOpenAI                    RelayFormat = "openai"
	RelayFormatClaude                                = "claude"
	RelayFormatGemini                                = "gemini"
	RelayFormatOpenAIResponses                       = "openai_responses"
	RelayFormatOpenAIResponsesCompaction             = "openai_responses_compaction"
	RelayFormatOpenAIAudio                           = "openai_audio"
	RelayFormatOpenAIImage                           = "openai_image"
	RelayFormatOpenAIRealtime                        = "openai_realtime"
	RelayFormatInteractions                          = "interactions"
	RelayFormatRerank                                = "rerank"
	RelayFormatEmbedding                             = "embedding"
	RelayFormatWebSearch                             = "web_search"

	RelayFormatTask    = "task"
	RelayFormatMjProxy = "mj_proxy"
)
