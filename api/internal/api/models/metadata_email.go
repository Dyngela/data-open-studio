package models

type MetadataEmail struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	Name     string `json:"name"`
	ImapHost string `json:"imapHost"`
	ImapPort int    `json:"imapPort" gorm:"default:993"`
	SmtpHost string `json:"smtpHost"`
	SmtpPort int    `json:"smtpPort" gorm:"default:587"`
	Username string `json:"username"`
	Password string `json:"password"`
	UseTLS   bool   `json:"useTls" gorm:"default:true"`
	Extra    string `json:"extra"`
}
