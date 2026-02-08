package service

import (
	"api"
	"api/internal/api/models"
	"api/internal/api/repo"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/rs/zerolog"
	gomail "github.com/wneessen/go-mail"
)

type EmailMessage struct {
	To          []string `json:"to"`
	CC          []string `json:"cc"`
	BCC         []string `json:"bcc"`
	Subject     string   `json:"subject"`
	Body        string   `json:"body"`
	IsHTML      bool     `json:"isHtml"`
	Attachments []string `json:"attachments"`
}

type MailService struct {
	logger    zerolog.Logger
	emailRepo repo.EmailMetadataRepository
}

func NewMailService() *MailService {
	return &MailService{
		logger:    api.Logger,
		emailRepo: *repo.NewEmailMetadataRepository(),
	}
}

// SendInternal sends an email using application-level SMTP config from .env.
// It uses SMTP_FROM as the sender address (falls back to SMTP_USERNAME).
func (s *MailService) SendInternal(msg EmailMessage) error {
	cfg := api.GetConfig().SmtpConfig
	if cfg.Host == "" || cfg.Username == "" {
		return fmt.Errorf("internal SMTP not configured (SMTP_HOST / SMTP_USERNAME missing)")
	}
	if len(msg.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	from := cfg.From
	if from == "" {
		from = cfg.Username
	}

	m := gomail.NewMsg()
	if err := m.From(from); err != nil {
		return fmt.Errorf("failed to set from: %w", err)
	}
	if err := m.To(msg.To...); err != nil {
		return fmt.Errorf("failed to set to: %w", err)
	}
	if len(msg.CC) > 0 {
		if err := m.Cc(msg.CC...); err != nil {
			return fmt.Errorf("failed to set cc: %w", err)
		}
	}
	if len(msg.BCC) > 0 {
		if err := m.Bcc(msg.BCC...); err != nil {
			return fmt.Errorf("failed to set bcc: %w", err)
		}
	}

	m.Subject(msg.Subject)
	if msg.IsHTML {
		m.SetBodyString(gomail.TypeTextHTML, msg.Body)
	} else {
		m.SetBodyString(gomail.TypeTextPlain, msg.Body)
	}

	tlsPolicy := gomail.TLSOpportunistic
	if cfg.UseTLS {
		tlsPolicy = gomail.TLSMandatory
	}

	opts := []gomail.Option{
		gomail.WithPort(cfg.Port),
		gomail.WithTLSPolicy(tlsPolicy),
	}
	if cfg.Password != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
			gomail.WithUsername(cfg.Username),
			gomail.WithPassword(cfg.Password),
		)
	}
	client, err := gomail.NewClient(cfg.Host, opts...)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if err := client.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info().Strs("to", msg.To).Str("subject", msg.Subject).Msg("Internal email sent")
	return nil
}

// IsInternalSmtpConfigured returns true when the app-level SMTP settings are filled in
func (s *MailService) IsInternalSmtpConfigured() bool {
	cfg := api.GetConfig().SmtpConfig
	return cfg.Host != "" && cfg.Username != ""
}

// SendWithMetadata loads MetadataEmail by ID and sends an email via SMTP
func (s *MailService) SendWithMetadata(metadataID uint, msg EmailMessage) error {
	var meta models.MetadataEmail
	if err := s.emailRepo.Db.First(&meta, metadataID).Error; err != nil {
		return fmt.Errorf("failed to load email metadata: %w", err)
	}

	return s.SendWithInline(meta.SmtpHost, meta.SmtpPort, meta.Username, meta.Password, meta.UseTLS, msg)
}

// SendWithInline sends an email using inline SMTP credentials
func (s *MailService) SendWithInline(host string, port int, username, password string, useTLS bool, msg EmailMessage) error {
	m := gomail.NewMsg()

	if err := m.From(username); err != nil {
		return fmt.Errorf("failed to set from: %w", err)
	}
	if err := m.To(msg.To...); err != nil {
		return fmt.Errorf("failed to set to: %w", err)
	}
	if len(msg.CC) > 0 {
		if err := m.Cc(msg.CC...); err != nil {
			return fmt.Errorf("failed to set cc: %w", err)
		}
	}
	if len(msg.BCC) > 0 {
		if err := m.Bcc(msg.BCC...); err != nil {
			return fmt.Errorf("failed to set bcc: %w", err)
		}
	}

	m.Subject(msg.Subject)

	if msg.IsHTML {
		m.SetBodyString(gomail.TypeTextHTML, msg.Body)
	} else {
		m.SetBodyString(gomail.TypeTextPlain, msg.Body)
	}

	tlsPolicy := gomail.TLSOpportunistic
	if useTLS {
		tlsPolicy = gomail.TLSMandatory
	}

	client, err := gomail.NewClient(host,
		gomail.WithPort(port),
		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
		gomail.WithUsername(username),
		gomail.WithPassword(password),
		gomail.WithTLSPolicy(tlsPolicy),
	)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if err := client.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info().Strs("to", msg.To).Str("subject", msg.Subject).Msg("Email sent successfully")
	return nil
}

// TestSMTPConnection tests an SMTP connection
func (s *MailService) TestSMTPConnection(host string, port int, username, password string, useTLS bool) error {
	tlsPolicy := gomail.TLSOpportunistic
	if useTLS {
		tlsPolicy = gomail.TLSMandatory
	}

	client, err := gomail.NewClient(host,
		gomail.WithPort(port),
		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
		gomail.WithUsername(username),
		gomail.WithPassword(password),
		gomail.WithTLSPolicy(tlsPolicy),
	)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := client.DialWithContext(ctx); err != nil {
		return fmt.Errorf("SMTP connection failed: %w", err)
	}
	_ = client.Close()

	return nil
}

// TestIMAPConnection tests an IMAP connection
func (s *MailService) TestIMAPConnection(host string, port int, username, password string, useTLS bool) error {
	addr := fmt.Sprintf("%s:%d", host, port)

	var client *imapclient.Client
	var err error
	if useTLS {
		client, err = imapclient.DialTLS(addr, &imapclient.Options{
			TLSConfig: &tls.Config{ServerName: host},
		})
	} else {
		client, err = imapclient.DialInsecure(addr, nil)
	}
	if err != nil {
		return fmt.Errorf("IMAP connection failed: %w", err)
	}
	defer client.Close()

	if err := client.Login(username, password).Wait(); err != nil {
		return fmt.Errorf("IMAP login failed: %w", err)
	}

	return nil
}
