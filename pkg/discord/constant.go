package discord

import "time"

const (
	webhookURLTemplate = "https://discord.com/api/webhooks/%s/%s"

	ColorBlue   = 3447003
	ColorGreen  = 3066993
	ColorYellow = 16776960
	ColorRed    = 15158332
	ColorPurple = 10181046
	ColorOrange = 15105570
	ColorGray   = 9807270
	ColorDark   = 0x36393F

	ColorInfo    = ColorBlue
	ColorSuccess = ColorGreen
	ColorWarning = ColorYellow
	ColorError   = ColorRed

	MaxMessageLength = 2000
	MaxEmbedLength    = 6000

	MaxTitleLen       = 256
	MaxDescriptionLen = 4096
	MaxFieldValueLen  = 1024
	ReportBugDescLen  = 4096
)

const (
	DefaultTimeout   = 30 * time.Second
	DefaultRetryCount = 3
	DefaultRetryDelay = 1 * time.Second
)

const (
	DefaultUsername  = "SMAP Bot"
	UserAgent        = "SMAP-Bot/1.0"
	ReportBugTitle   = "SMAP Service Error Report"
	ActivityLogTitle = "Activity Log"
)
