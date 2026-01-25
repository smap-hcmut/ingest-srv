package email

import "embed"

//go:embed templates/*
var emailTemplates embed.FS

const (
	EmailVerificationTemplate = "email_verification"
)
