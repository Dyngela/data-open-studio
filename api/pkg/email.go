package pkg

import (
	"api"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	auth "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

var providerRegistry = make(map[EmailProvider]IEmailProvider)
var defaultPriority []EmailProvider

const maxRetries = 2

func InitializeEmailsProviders() {
	brevo := &BrevoProvider{}
	brevo.init()

	smtp := &SMTPCustomProvider{}
	smtp.init()

	outlook := &OutlookProvider{}
	outlook.init()

	providerRegistry[EMAIL_PROVIDER_BREVO] = brevo
	providerRegistry[EMAIL_PROVIDER_SMTP_CUSTOM] = smtp
	providerRegistry[EMAIL_PROVIDER_OUTLOOK] = outlook

	defaultPriority = append(defaultPriority, brevo.name(), outlook.name(), smtp.name())
}

type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

func AttachmentFromFile(path string) (Attachment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Attachment{}, err
	}
	return Attachment{
		Filename:    filepath.Base(path),
		ContentType: http.DetectContentType(data),
		Data:        data,
	}, nil
}

func AttachmentFromCSV(filename string, csvData string) Attachment {
	return Attachment{
		Filename:    filename,
		ContentType: "text/csv",
		Data:        []byte(csvData),
	}
}

func AttachmentFromBase64(filename string, contentType string, b64String string) (Attachment, error) {
	data, err := base64.StdEncoding.DecodeString(b64String)
	if err != nil {
		return Attachment{}, fmt.Errorf("invalid base64: %w", err)
	}
	return Attachment{
		Filename:    filename,
		ContentType: contentType,
		Data:        data,
	}, nil
}

type EmailMessage struct {
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	Body        string
	IsHTML      bool
	Attachments []Attachment // Changed from []string
}

type IEmailProvider interface {
	init()
	isInitialized() bool
	send(msg EmailMessage) iCustomEmailError
	name() EmailProvider // Useful for logging/metrics
}

type iCustomEmailError interface {
	error
	Temporary() bool
}

type CustomEmailError struct {
	Msg  string
	Temp bool
}

func (e *CustomEmailError) Error() string   { return e.Msg }
func (e *CustomEmailError) Temporary() bool { return e.Temp }

func SendEmail(msg EmailMessage, requestedProviders ...EmailProvider) error {
	if len(requestedProviders) == 0 {
		requestedProviders = defaultPriority
	}

	var errs []string

	for _, providerID := range requestedProviders {
		impl, exists := providerRegistry[providerID]
		if !exists || !impl.isInitialized() {
			errs = append(errs, fmt.Sprintf("provider %v: skipped (not ready)", providerID))
			continue
		}

		var lastErr iCustomEmailError
		for i := 0; i < maxRetries; i++ {
			lastErr = impl.send(msg)

			if lastErr == nil {
				return nil // Success!
			}

			// Check if we should stop immediately (Permanent Error)
			if !lastErr.Temporary() {
				return fmt.Errorf("permanent error from %v: %w", providerID, lastErr)
			}

			if i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
			}
		}

		errs = append(errs, fmt.Sprintf("%v after %d attempts: %v", providerID, maxRetries, lastErr))
	}

	return fmt.Errorf("all email providers failed: %s", strings.Join(errs, " | "))
}

type EmailProvider int

const (
	EMAIL_PROVIDER_BREVO EmailProvider = iota
	EMAIL_PROVIDER_SMTP_CUSTOM
	EMAIL_PROVIDER_OUTLOOK
	EMAIL_PROVIDER_DEFAULT = EMAIL_PROVIDER_BREVO
)

type BrevoProvider struct {
	apiKey      string
	initialized bool
}

func (b *BrevoProvider) init() {
	// In real usage: b.apiKey = os.Getenv("BREVO_API_KEY")
	b.initialized = true
}
func (b *BrevoProvider) isInitialized() bool { return b.initialized }
func (b *BrevoProvider) send(msg EmailMessage) iCustomEmailError {
	fmt.Printf("Sending via Brevo to %v\n", msg.To)
	return nil
}
func (b *BrevoProvider) name() EmailProvider { return EMAIL_PROVIDER_BREVO }

type SMTPCustomProvider struct {
	Host        string
	Port        int
	initialized bool
}

func (s *SMTPCustomProvider) isInitialized() bool {
	return s.initialized
}
func (s *SMTPCustomProvider) init() {
	s.initialized = true
}
func (s *SMTPCustomProvider) send(msg EmailMessage) iCustomEmailError {
	// Implement standard SMTP logic here
	return nil
}
func (s *SMTPCustomProvider) name() EmailProvider { return EMAIL_PROVIDER_SMTP_CUSTOM }

type OutlookProvider struct {
	initialized            bool
	clientSecretCredential *azidentity.ClientSecretCredential
	userClient             *msgraphsdk.GraphServiceClient
	graphUserScopes        []string
	config                 api.AppConfig
}

func (slf *OutlookProvider) init() {
	slf.config = api.GetConfig()
	clientId := slf.config.SMTP.Outlook.OutlookClientID
	clientSecret := slf.config.SMTP.Outlook.OutlookClientSecret
	tenantId := slf.config.SMTP.Outlook.OutlookTenantID

	if clientId == "" || clientSecret == "" || tenantId == "" {
		slf.initialized = false
		return
	}

	slf.graphUserScopes = []string{"https://graph.microsoft.com/.default"}

	credential, err := azidentity.NewClientSecretCredential(
		tenantId,
		clientId,
		clientSecret,
		nil,
	)
	if err != nil {
		slf.initialized = false
		return
	}

	slf.clientSecretCredential = credential

	authProvider, err := auth.NewAzureIdentityAuthenticationProviderWithScopes(credential, slf.graphUserScopes)
	if err != nil {
		slf.initialized = false
		return
	}

	adapter, err := msgraphsdk.NewGraphRequestAdapter(authProvider)
	if err != nil {
		slf.initialized = false
		return
	}

	client := msgraphsdk.NewGraphServiceClient(adapter)
	slf.userClient = client
	slf.initialized = true
}

func (slf *OutlookProvider) isInitialized() bool                     { return slf.initialized }
func (slf *OutlookProvider) send(msg EmailMessage) iCustomEmailError { return nil }
func (slf *OutlookProvider) name() EmailProvider                     { return EMAIL_PROVIDER_OUTLOOK }
