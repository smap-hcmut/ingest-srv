package locale

import (
	"context"
	"strings"
)

// ParseLang parses and validates a language code. Returns DefaultLang if not supported.
// Input is case-insensitive and trimmed.
func ParseLang(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	switch lang {
	case EN, "english":
		return EN
	case VI, "vietnamese", "việt nam":
		return VI
	case JA, "japanese":
		return JA
	default:
		return DefaultLang
	}
}

// IsValidLang reports whether the language code is supported.
func IsValidLang(lang string) bool {
	lang = strings.TrimSpace(strings.ToLower(lang))
	for _, supported := range LangList {
		if lang == supported {
			return true
		}
	}
	return false
}

// GetLang returns the locale from context, or DefaultLang if not set.
func GetLang(ctx context.Context) string {
	lang, ok := GetLocaleFromContext(ctx)
	if !ok {
		return DefaultLang
	}
	return lang
}

// SetLocaleToContext sets the locale in the context. Invalid lang is replaced with DefaultLang.
func SetLocaleToContext(ctx context.Context, lang string) context.Context {
	if !IsValidLang(lang) {
		lang = DefaultLang
	}
	return context.WithValue(ctx, Locale{}, lang)
}

// GetLocaleFromContext returns the locale from context. Second return is false if not set or empty.
func GetLocaleFromContext(ctx context.Context) (string, bool) {
	lang, ok := ctx.Value(Locale{}).(string)
	if !ok || lang == "" {
		return "", false
	}
	return lang, true
}
