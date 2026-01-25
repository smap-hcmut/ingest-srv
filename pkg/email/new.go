package email

import (
	"bytes"
	"context"

	"smap-collector/pkg/locale"
)

func NewEmail(ctx context.Context, e EmailMeta, data interface{}) (Email, error) {
	l, ok := locale.GetLocaleFromContext(ctx)
	if !ok {
		if data, ok := ctx.Value(locale.Locale{}).(string); ok {
			l = data
		} else {
			return Email{}, locale.ErrLocaleNotFound
		}
	}

	tmpl, err := getEmailTemplate(l, e.TemplateType)
	if err != nil {
		return Email{}, err
	}
	translatedData := map[string]interface{}{}
	translateData(l, e.TemplateType, data, &translatedData)

	var body bytes.Buffer
	if err := tmpl.Execute(&body, translatedData); err != nil {
		return Email{}, err
	}

	return Email{
		Recipient: e.Recipient,
		CC:        e.CC,
		Subject:   getEmailSubject(l, e.TemplateType),
		Body:      body.String(),
	}, nil
}
