package email

type EmailMeta struct {
	Recipient    string
	CC           []string
	TemplateType string
}
type Email struct {
	Recipient string
	Subject   string
	Body      string
	CC        []string
}

// These types are used to apply data to email templates
type EmailVerification struct {
	Name         string
	Email        string
	OTP          string
	OTPExpireMin string
}
