package service

import (
	"api"
	"api/internal/api/repo"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/rs/zerolog"
	gomail "github.com/wneessen/go-mail"
)

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

// TestSMTPConnection tests an SMTP connection
func (slf *MailService) TestSMTPConnection(host string, port int, username, password string, useTLS bool) error {
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
func (slf *MailService) TestIMAPConnection(host string, port int, username, password string, useTLS bool) error {
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
