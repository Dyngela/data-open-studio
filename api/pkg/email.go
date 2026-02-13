package pkg

import (
	"api"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/getbrevo/brevo-go/lib"
	auth "github.com/microsoft/kiota-authentication-azure-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	msgraphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/rs/zerolog"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/microsoftgraph/msgraph-sdk-go/users"
)

var providerRegistry = make(map[EmailProvider]IEmailProvider)
var defaultPriority []EmailProvider

const maxRetries = 2

func InitializeEmailsProviders(logger zerolog.Logger) {
	brevo := &BrevoProvider{
		logger: logger,
	}
	brevo.init()

	smtp := &SMTPCustomProvider{
		logger: logger,
	}
	smtp.init()

	outlook := &OutlookProvider{
		logger: logger,
	}
	outlook.init()

	providerRegistry[EMAIL_PROVIDER_BREVO] = brevo
	providerRegistry[EMAIL_PROVIDER_OUTLOOK] = outlook
	providerRegistry[EMAIL_PROVIDER_SMTP_CUSTOM] = smtp

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
	logger      zerolog.Logger
	config      api.AppConfig
	initialized bool
	client      *lib.APIClient
}

func (slf *BrevoProvider) init() {
	config := api.GetConfig()
	key := config.SMTP.Brevo.APIKey
	if key == "" {
		slf.initialized = false
		return
	}
	cfg := lib.NewConfiguration()
	cfg.AddDefaultHeader("api-key", key)
	slf.client = lib.NewAPIClient(cfg)
	slf.initialized = true
}

func (slf *BrevoProvider) isInitialized() bool { return slf.initialized }

func (slf *BrevoProvider) send(msg EmailMessage) iCustomEmailError {
	var html string
	var text string

	if msg.IsHTML {
		html = msg.Body
	} else {
		text = msg.Body
	}

	var to []lib.SendSmtpEmailTo
	var cc []lib.SendSmtpEmailCc
	var bcc []lib.SendSmtpEmailBcc
	var attachments []lib.SendSmtpEmailAttachment

	for _, val := range msg.To {
		to = append(to, lib.SendSmtpEmailTo{Email: val})
	}
	for _, val := range msg.CC {
		cc = append(cc, lib.SendSmtpEmailCc{Email: val})
	}
	for _, val := range msg.BCC {
		bcc = append(bcc, lib.SendSmtpEmailBcc{Email: val})
	}
	for _, val := range msg.Attachments {
		attachments = append(attachments, lib.SendSmtpEmailAttachment{
			Content: base64.StdEncoding.EncodeToString(val.Data),
			Name:    val.Filename,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, resp, err := slf.client.TransactionalEmailsApi.SendTransacEmail(ctx, lib.SendSmtpEmail{
		Sender: &lib.SendSmtpEmailSender{
			Email: slf.config.SMTP.SenderEmail,
		},
		To:          to,
		Bcc:         bcc,
		Cc:          cc,
		HtmlContent: html,
		TextContent: text,
		Subject:     msg.Subject,
		Attachment:  nil,
	})
	if resp == nil {
		slf.logger.Error().Msg("Brevo send error: nil response")
		return &CustomEmailError{Msg: "Brevo send error: nil response", Temp: true}
	}

	if err != nil {
		slf.logger.Error().Err(err).Msg("Brevo send error")
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		return nil
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		return &CustomEmailError{Msg: fmt.Sprintf("Brevo send error: %v", resp.Status), Temp: false}
	default:
		return &CustomEmailError{Msg: fmt.Sprintf("Brevo send error: %v", resp.Status), Temp: true}
	}
}

func (slf *BrevoProvider) name() EmailProvider { return EMAIL_PROVIDER_BREVO }

type SMTPCustomProvider struct {
	logger zerolog.Logger

	Host        string
	Port        int
	initialized bool
}

func (s *SMTPCustomProvider) isInitialized() bool {
	return s.initialized
}

func (s *SMTPCustomProvider) init() {
	s.initialized = false
}

func (s *SMTPCustomProvider) send(msg EmailMessage) iCustomEmailError {
	// Implement standard SMTP logic here
	return nil
}

func (s *SMTPCustomProvider) name() EmailProvider { return EMAIL_PROVIDER_SMTP_CUSTOM }

type OutlookProvider struct {
	logger zerolog.Logger

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
	email := slf.config.SMTP.SenderEmail
	objectId := slf.config.SMTP.Outlook.SenderAzureObjectID

	if clientId == "" || clientSecret == "" || tenantId == "" || email == "" || objectId == "" {
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

func (slf *OutlookProvider) isInitialized() bool { return slf.initialized }

func (slf *OutlookProvider) send(msg EmailMessage) iCustomEmailError {
	message := msgraphmodels.NewMessage()
	message.SetSubject(ToPtr(msg.Subject))
	message.SetBody(msgraphmodels.NewItemBody())

	var bodyType *models.BodyType
	if msg.IsHTML {
		bodyType = ToPtr(models.HTML_BODYTYPE)
	} else {
		bodyType = ToPtr(models.TEXT_BODYTYPE)
	}

	message.GetBody().SetContentType(bodyType)
	message.GetBody().SetContent(ToPtr(msg.Body))

	fromEmail := msgraphmodels.NewRecipient()
	fromEmailAddress := msgraphmodels.NewEmailAddress()
	fromEmailAddress.SetAddress(ToPtr(slf.config.SMTP.SenderEmail))
	fromEmail.SetEmailAddress(fromEmailAddress)
	message.SetFrom(fromEmail)
	if len(msg.To)+len(msg.CC)+len(msg.BCC) == 0 {
		return &CustomEmailError{Msg: "No recipients specified", Temp: false}
	}

	var to []msgraphmodels.Recipientable
	for _, val := range msg.To {
		recipient := msgraphmodels.NewRecipient()
		emailAddress := msgraphmodels.NewEmailAddress()
		emailAddress.SetAddress(ToPtr(val))
		recipient.SetEmailAddress(emailAddress)
		to = append(to, recipient)
	}
	message.SetToRecipients(to)

	var ccRecipients []msgraphmodels.Recipientable
	for _, val := range msg.CC {
		recipient := msgraphmodels.NewRecipient()
		emailAddress := msgraphmodels.NewEmailAddress()
		emailAddress.SetAddress(ToPtr(val))
		recipient.SetEmailAddress(emailAddress)
		ccRecipients = append(ccRecipients, recipient)

	}
	message.SetCcRecipients(ccRecipients)

	var bccRecipients []msgraphmodels.Recipientable
	for _, val := range msg.BCC {
		recipient := msgraphmodels.NewRecipient()
		emailAddress := msgraphmodels.NewEmailAddress()
		emailAddress.SetAddress(ToPtr(val))
		recipient.SetEmailAddress(emailAddress)
		bccRecipients = append(bccRecipients, recipient)
	}
	message.SetBccRecipients(bccRecipients)

	var msAttachments []msgraphmodels.Attachmentable
	for _, attachment := range msg.Attachments {
		fileAttachment := msgraphmodels.NewFileAttachment()
		fileAttachment.SetName(ToPtr(attachment.Filename))
		fileAttachment.SetContentBytes(attachment.Data)
		fileAttachment.SetSize(ToPtr(int32(len(attachment.Data))))
		fileAttachment.SetContentType(ToPtr(attachment.ContentType))
		msAttachments = append(msAttachments, fileAttachment)
	}

	message.SetAttachments(msAttachments)
	sendMailRequest := users.NewItemSendMailPostRequestBody()
	sendMailRequest.SetMessage(message)
	sendMailRequest.SetSaveToSentItems(ToPtr(false))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := slf.userClient.
		Users().
		ByUserId(slf.config.SMTP.Outlook.SenderAzureObjectID).
		SendMail().
		Post(ctx, sendMailRequest, nil)
	if err != nil {
		return &CustomEmailError{Msg: fmt.Sprintf("Outlook send error: %v", err), Temp: true}
	}

	return nil

}

func (slf *OutlookProvider) name() EmailProvider { return EMAIL_PROVIDER_OUTLOOK }
