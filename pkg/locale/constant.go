package locale

const (
	// EN is English.
	EN = "en"
	// VI is Vietnamese.
	VI = "vi"
	// JA is Japanese.
	JA = "ja"
)

// LangList contains all supported language codes.
var LangList = []string{EN, VI, JA}

// DefaultLang is the default language when no valid locale is provided.
var DefaultLang = EN
