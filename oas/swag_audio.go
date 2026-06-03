package oas

// AudioTranscriptions godoc
// @Summary 语音识别
// @Description 将音频文件转录为文字(ASR)
// @Description
// @Description * OpenAI: https://developers.openai.com/api/reference/resources/audio/subresources/transcriptions/methods/create
// @Description * 豆包ASR: https://www.volcengine.com/docs/6561/1354868
// @Description * 阿里ASR: https://help.aliyun.com/zh/model-studio/qwen-asr-api-reference
// @Description
// @Description **示例**:
// @Description ```bash
// @Description --form 'file=@welcome.mp3'
// @Description --form 'model=whisper-1'
// @Description --form 'prompt=简体'
// @Description ```
// @Tags OpenAI
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "需要转录的音频文件，支持格式：m4a, mp3, webm, mp4, mpga, wav, mpeg。文件大小最大 25 MB"
// @Param model formData string true "模型名称" example(gpt-4o-transcribe)
// @Param language formData string false "音频的语言(可选)，支持 ISO-639-1 格式"
// @Param prompt formData string false "可选的提示文本，用于指导模型的风格或继续前面的音频片段" example(简体)
// //@Param response_format formData string false "转录输出的格式" Enums(,json,text,srt,verbose_json,vtt)
// //@Param temperature formData number false "采样温度，介于 0 和 1 之间" example(0.5)
// //@Param timestamp_granularities[] formData string false "时间戳粒度" Enums(,word,segment)
// @Success 200 {object} dto.AudioResponse
// @Router /v1/audio/transcriptions [post]
func AudioTranscriptions() {
	// need `sudo apt install ffmpeg`
}

// AudioSpeech godoc
// @Summary 语音生成
// @Description 将文本转换为语音TTS
// @Description
// @Description * OpenAI: https://platform.openai.com/docs/guides/text-to-speech
// @Description * 豆包语音: https://www.volcengine.com/docs/6561/1257584
// @Description * 豆包音色: https://www.volcengine.com/docs/6561/1257544
// @Description * MiniMax 语音: https://platform.minimax.io/docs/api-reference/speech-t2a-http
// @Description
// @Description **示例**:
// @Description ```json
// @Description {
// @Description   "model": "tts-1",
// @Description   "input": "The quick brown fox jumped over the lazy dog.",
// @Description   "voice": "alloy",
// @Description   "speed": 1.0,
// @Description   "response_format": "mp3"
// @Description }
// @Description ```
// @Tags OpenAI
// @Accept json
// @Produce application/octet-stream
// @Param request body dto.AudioRequest true "TTS请求参数"
// @Success 200 {file} binary "音频文件流（根据 response_format 返回 mp3/opus/aac/flac/wav/pcm 格式）"
// @Router /v1/audio/speech [post]
func AudioSpeech() {
}

// AudioTranslations godoc
// @Summary 音频翻译
// @Description 将音频文件翻译为英文
// @Description
// @Description **文档**: https://developers.openai.com/api/reference/resources/audio/subresources/translations/methods/create
// @Description
// @Description **示例**:
// @Description ```bash
// @Description --form 'file=@german.mp3'
// @Description --form 'model=whisper-1'
// @Description --form 'prompt=translate to english'
// @Description ```
// @Tags OpenAI
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "需要翻译的音频文件，支持格式：m4a, mp3, webm, mp4, mpga, wav, mpeg。文件大小最大 25 MB"
// @Param model formData string true "模型名称" example(gpt-4o-transcribe)
// @Param prompt formData string false "可选的提示文本，用于指导模型的风格" example(translate to english)
// //@Param response_format formData string false "翻译输出的格式" Enums(,json,text,srt,verbose_json,vtt)
// //@Param temperature formData number false "采样温度，介于 0 和 1 之间" example(0.5)
// @Success 200 {object} dto.AudioResponse
// @Router /v1/audio/translations [post]
//func AudioTranslations() {
//}
