package tgbotidparse

type FileType int

const (
	// Thumbnail 缩略图
	Thumbnail FileType = iota
	// ProfilePhoto 用户和频道的个人照片，聊天照片是普通的照片
	ProfilePhoto
	// Photo 普通照片
	Photo
	// Voice 语音消息
	Voice
	// Video 视频
	Video
	// Document 文档
	Document
	// Encrypted 密聊文档
	Encrypted
	// Temp 临时文档
	Temp
	// Sticker 贴纸
	Sticker
	// Audio 音频
	Audio
	// Animation GIF动画
	Animation
	// EncryptedThumbnail 加密缩略图
	EncryptedThumbnail
	// Wallpaper 壁纸
	Wallpaper
	// VideoNote 圆形视频
	VideoNote
	// SecureRaw 护照原始文件
	SecureRaw
	// Secure 护照文件
	Secure
	// Background 背景
	Background
	// Size 大小
	Size
	// None 无
	None
)

var typeMap = map[FileType]string{
	Thumbnail:          "thumbnail",
	ProfilePhoto:       "profile_photo",
	Photo:              "photo",
	Voice:              "voice",
	Video:              "video",
	Document:           "document",
	Encrypted:          "encrypted",
	Temp:               "temp",
	Sticker:            "sticker",
	Audio:              "audio",
	Animation:          "animation",
	EncryptedThumbnail: "encrypted_thumbnail",
	Wallpaper:          "wallpaper",
	VideoNote:          "video_note",
	SecureRaw:          "secure_raw",
	Secure:             "secure",
	Background:         "background",
	Size:               "size",
	None:               "none", // 添加了 NONE 常量
}

const (
	UniqueWeb       = 0
	UniquePhoto     = 1
	UniqueDocument  = 2
	UniqueSecure    = 3
	UniqueEncrypted = 4
	UniqueTemp      = 5
)

var (
	UniqueTypes = map[int]string{
		UniqueWeb:       "web",
		UniquePhoto:     "photo",
		UniqueDocument:  "document",
		UniqueSecure:    "secure",
		UniqueEncrypted: "encrypted",
		UniqueTemp:      "temp",
	}

	UniqueTypesIDs = map[string]int{
		"web":       UniqueWeb,
		"photo":     UniquePhoto,
		"document":  UniqueDocument,
		"secure":    UniqueSecure,
		"encrypted": UniqueEncrypted,
		"temp":      UniqueTemp,
	}

	FullUniqueMap = map[FileType]int{
		Photo:              UniquePhoto,
		ProfilePhoto:       UniquePhoto,
		Thumbnail:          UniquePhoto,
		EncryptedThumbnail: UniquePhoto,
		Wallpaper:          UniquePhoto,
		Video:              UniqueDocument,
		Voice:              UniqueDocument,
		Document:           UniqueDocument,
		Sticker:            UniqueDocument,
		Audio:              UniqueDocument,
		Animation:          UniqueDocument,
		VideoNote:          UniqueDocument,
		Background:         UniqueDocument,
		Secure:             UniqueSecure,
		SecureRaw:          UniqueSecure,
		Encrypted:          UniqueEncrypted,
		Temp:               UniqueTemp,
	}
)

const (
	PhotoSizeSourceLegacyC                    = 0
	PhotoSizeSourceThumbnailC                 = 1
	PhotoSizeSourceDialogPhotoSmall           = 2
	PhotoSizeSourceDialogPhotoBig             = 3
	PhotoSizeSourceStickerSetThumbnail        = 4
	PhotoSizeSourceFullLegacy                 = 5
	PhotoSizeSourceDialogPhotoSmallLegacy     = 6
	PhotoSizeSourceDialogPhotoBigLegacy       = 7
	PhotoSizeSourceStickerSetThumbnailLegacy  = 8
	PhotoSizeSourceStickerSetThumbnailVersion = 9
	WebLocationFlag                           = 1 << 24
	FileReferenceFlag                         = 1 << 25
)
