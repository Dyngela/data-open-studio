package models

// EmailOutputConfig holds configuration for email_output nodes
type EmailOutputConfig struct {
	// Reference to MetadataEmail (alternative to inline credentials)
	MetadataEmailID *uint `json:"metadataEmailId,omitempty"`

	// Inline SMTP settings (used if MetadataEmailID is nil)
	SmtpHost string `json:"smtpHost,omitempty"`
	SmtpPort int    `json:"smtpPort,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	UseTLS   bool   `json:"useTls,omitempty"`

	// Email fields (support Go template syntax for per-row rendering)
	To      []string `json:"to"`
	CC      []string `json:"cc,omitempty"`
	BCC     []string `json:"bcc,omitempty"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	IsHTML  bool     `json:"isHtml"`
}
