package constant

type TaskPlatform string

const (
	TaskPlatformSuno       TaskPlatform = "suno"
	TaskPlatformMidjourney              = "mj"
)

const (
	SunoActionMusic       = "MUSIC"
	SunoActionLyrics      = "LYRICS"
	SunoActionUploadCover = "UPLOAD-COVER"

	TaskActionGenerate          = "generate"
	TaskActionTextGenerate      = "textGenerate"
	TaskActionFirstTailGenerate = "firstTailGenerate"
	TaskActionReferenceGenerate = "referenceGenerate"
	TaskActionRemix             = "remixGenerate"
	TaskActionMultiFrame        = "multiFrameGenerate" // vidu智能多帧
	TaskActionMotionControl     = "motionControl"      // kling 动作控制
)

var SunoModel2Action = map[string]string{
	"suno_music":        SunoActionMusic,
	"suno_lyrics":       SunoActionLyrics,
	"suno_upload-cover": SunoActionUploadCover,
}
