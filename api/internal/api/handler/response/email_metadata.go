package response

type EmailMetadata struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	ImapHost string `json:"imapHost"`
	ImapPort int    `json:"imapPort"`
	SmtpHost string `json:"smtpHost"`
	SmtpPort int    `json:"smtpPort"`
	Username string `json:"username"`
	Password string `json:"password"`
	UseTLS   bool   `json:"useTls"`
	Extra    string `json:"extra"`
}

type TestEmailConnectionResult struct {
	ImapSuccess bool   `json:"imapSuccess"`
	ImapMessage string `json:"imapMessage"`
	SmtpSuccess bool   `json:"smtpSuccess"`
	SmtpMessage string `json:"smtpMessage"`
}
