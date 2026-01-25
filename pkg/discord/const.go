package discord

const (
	baseURL    = "https://discord.com/identity0"
	webhookURL = "https://discord.com/api/webhooks/%s/%s"

	// Discord Embed Colors
	ColorBlue   = 3447003
	ColorGreen  = 3066993
	ColorYellow = 16776960
	ColorRed    = 15158332
	ColorPurple = 10181046
	ColorOrange = 15105570
	ColorGray   = 9807270
	ColorDark   = 0x36393F

	// Default colors for message types
	ColorInfo    = ColorBlue
	ColorSuccess = ColorGreen
	ColorWarning = ColorYellow
	ColorError   = ColorRed

	// Max message length
	MaxMessageLength = 2000
	MaxEmbedLength   = 6000
)
