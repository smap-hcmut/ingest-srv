package locale

import (
	"context"
	"errors"
)

type Locale struct{}

var ErrLocaleNotFound = errors.New("locale not found")

var LangList = []string{"en", "vi", "ja"}

const (
	EN = "en"
	VI = "vi"
	JA = "ja"
)

var DefaultLang = EN

func ParseLang(lang string) string {
	switch lang {
	case "en":
		return EN
	case "vi":
		return VI
	case "ja":
		return JA
	default:
		return DefaultLang
	}
}

func GetLang(ctx context.Context) string {
	lang, ok := GetLocaleFromContext(ctx)
	if !ok {
		lang = DefaultLang
	}

	return lang
}

func SetLocaleToContext(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, Locale{}, lang)
}

func GetLocaleFromContext(ctx context.Context) (string, bool) {
	l, ok := ctx.Value(Locale{}).(string)
	return l, ok
}
