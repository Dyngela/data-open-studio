package websocket

import (
	"api"
	"api/internal/api/models"
	"api/internal/api/service"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
)

// MessageProcessor handles WebSocket messages and performs database operations
type MessageProcessor struct {
	jobService          *service.JobService
	metadataService     *service.MetadataService
	sftpMetadataService *service.SftpMetadataService
	logger              zerolog.Logger
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor() *MessageProcessor {
	return &MessageProcessor{
		jobService:          service.NewJobService(),
		metadataService:     service.NewMetadataService(),
		sftpMetadataService: service.NewSftpMetadataService(),
		logger:              api.Logger,
	}
}

// ProcessMessage processes a message and performs necessary database operations
// Returns the updated message to broadcast, or error if processing failed
func (p *MessageProcessor) ProcessMessage(msg *Message) (*Message, error) {
	p.logger.Debug().Msgf("Processing message: %+v", msg)
	switch msg.Type {
	case MessageTypeJobUpdate:
		return p.processJobUpdate(msg)
	case MessageTypeJobDelete:
		return p.processJobDelete(msg)
	case MessageTypeJobCreate:
		return p.processJobCreate(msg)
	case MessageTypeJobExecute:
		return p.processJobExecute(msg)

	case MessageTypeDbMetadataGet:
		return p.processDbMetadataGet(msg)
	case MessageTypeDbMetadataGetAll:
		return p.processDbMetadataGetAll(msg)
	case MessageTypeDbMetadataUpdate:
		return p.processDbMetadataUpdate(msg)
	case MessageTypeDbMetadataDelete:
		return p.processDbMetadataDelete(msg)
	case MessageTypeDbMetadataCreate:
		return p.processDbMetadataCreate(msg)

	case MessageTypeSftpMetadataGet:
		return p.processSftpMetadataGet(msg)
	case MessageTypeSftpMetadataGetAll:
		return p.processSftpMetadataGetAll(msg)
	case MessageTypeSftpMetadataUpdate:
		return p.processSftpMetadataUpdate(msg)
	case MessageTypeSftpMetadataDelete:
		return p.processSftpMetadataDelete(msg)
	case MessageTypeSftpMetadataCreate:
		return p.processSftpMetadataCreate(msg)

	case MessageTypeDbNodeGuessDataModel:
		return p.processDbNodeGuessDataModel(msg)
	default:
		// Other message types don't require processing (chat, cursor, etc.)

		panic("unsupported message type: " + string(msg.Type))
	}
}

func (p *MessageProcessor) validateData(msg *Message, out any) error {
	dataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal message data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, out); err != nil {
		return fmt.Errorf("invalid message data: %w", err)
	}

	return nil
}

// processJobUpdate handles updating job metadata
func (p *MessageProcessor) processJobUpdate(msg *Message) (*Message, error) {
	var jobData JobUpdate
	if err := p.validateData(msg, &jobData); err != nil {
		return nil, err
	}

	var patch = make(map[string]any)

	if jobData.Name != nil {
		patch["name"] = *jobData.Name
	}
	if jobData.Description != nil {
		patch["description"] = *jobData.Description
	}
	if jobData.Active != nil {
		patch["active"] = *jobData.Active
	}
	if jobData.Nodes != nil {
		nodesBytes, err := json.Marshal(jobData.Nodes)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal nodes: %w", err)
		}
		var nodeList []models.Node
		if err := json.Unmarshal(nodesBytes, &nodeList); err != nil {
			return nil, fmt.Errorf("failed to unmarshal nodes: %w", err)
		}
		patch["nodes"] = nodeList
	}

	if err := p.jobService.Patch(patch); err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	p.logger.Info().
		Uint("jobId", msg.JobID).
		Uint("userId", msg.UserID).
		Msg("Job updated via WebSocket")

	return msg, nil
}

func (p *MessageProcessor) processJobDelete(msg *Message) (*Message, error) {
	if err := p.jobService.DeleteJob(msg.JobID); err != nil {
		return nil, err
	}
	return msg, nil
}

func (p *MessageProcessor) processJobCreate(msg *Message) (*Message, error) {
	return msg, nil
}
func (p *MessageProcessor) processJobExecute(msg *Message) (*Message, error) {
	return msg, nil
}
func (p *MessageProcessor) processDbMetadataGet(msg *Message) (*Message, error) {
	var data struct {
		ID uint `json:"id"`
	}
	if err := p.validateData(msg, &data); err != nil {
		return nil, err
	}

	metadata, err := p.metadataService.FindByID(data.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get db metadata: %w", err)
	}

	msg.Data = metadata
	msg.Type = ResponseDbMetadataGet

	p.logger.Info().
		Uint("metadataId", data.ID).
		Uint("userId", msg.UserID).
		Msg("DB MetadataDatabase retrieved via WebSocket")

	return msg, nil
}

func (p *MessageProcessor) processDbMetadataGetAll(msg *Message) (*Message, error) {
	metadataList, err := p.metadataService.FindAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all db metadata: %w", err)
	}

	msg.Data = metadataList
	msg.Type = ResponseDbMetadataGetAll

	p.logger.Info().
		Uint("userId", msg.UserID).
		Int("count", len(metadataList)).
		Msg("All DB MetadataDatabase retrieved via WebSocket")

	return msg, nil
}

func (p *MessageProcessor) processDbMetadataCreate(msg *Message) (*Message, error) {
	var data struct {
		Host         string `json:"host"`
		Port         string `json:"port"`
		User         string `json:"user"`
		Password     string `json:"password"`
		DatabaseName string `json:"databaseName"`
		SSLMode      string `json:"sslMode"`
		Extra        string `json:"extra"`
	}
	if err := p.validateData(msg, &data); err != nil {
		return nil, err
	}

	metadata := models.MetadataDatabase{
		Host:         data.Host,
		Port:         data.Port,
		User:         data.User,
		Password:     data.Password,
		DatabaseName: data.DatabaseName,
		SSLMode:      data.SSLMode,
		Extra:        data.Extra,
	}

	createdMetadata, err := p.metadataService.Create(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create db metadata: %w", err)
	}

	msg.Data = createdMetadata
	msg.Type = ResponseDbMetadataCreate

	p.logger.Info().
		Uint("metadataId", createdMetadata.ID).
		Uint("userId", msg.UserID).
		Msg("DB MetadataDatabase created via WebSocket")

	return msg, nil
}

func (p *MessageProcessor) processDbMetadataDelete(msg *Message) (*Message, error) {
	var data struct {
		ID uint `json:"id"`
	}
	if err := p.validateData(msg, &data); err != nil {
		return nil, err
	}

	if err := p.metadataService.Delete(data.ID); err != nil {
		return nil, fmt.Errorf("failed to delete db metadata: %w", err)
	}

	msg.Data = map[string]any{
		"id":      data.ID,
		"deleted": true,
	}
	msg.Type = ResponseDbMetadataDelete

	p.logger.Info().
		Uint("metadataId", data.ID).
		Uint("userId", msg.UserID).
		Msg("DB MetadataDatabase deleted via WebSocket")

	return msg, nil
}

func (p *MessageProcessor) processDbMetadataUpdate(msg *Message) (*Message, error) {
	var data struct {
		ID           uint    `json:"id"`
		Host         *string `json:"host,omitempty"`
		Port         *string `json:"port,omitempty"`
		User         *string `json:"user,omitempty"`
		Password     *string `json:"password,omitempty"`
		DatabaseName *string `json:"databaseName,omitempty"`
		SSLMode      *string `json:"sslMode,omitempty"`
	}
	if err := p.validateData(msg, &data); err != nil {
		return nil, err
	}

	if data.ID == 0 {
		return nil, fmt.Errorf("metadata ID is required for update")
	}

	patch := make(map[string]any)
	if data.Host != nil {
		patch["host"] = *data.Host
	}
	if data.Port != nil {
		patch["port"] = *data.Port
	}
	if data.User != nil {
		patch["user"] = *data.User
	}
	if data.Password != nil {
		patch["password"] = *data.Password
	}
	if data.DatabaseName != nil {
		patch["database_name"] = *data.DatabaseName
	}
	if data.SSLMode != nil {
		patch["ssl_mode"] = *data.SSLMode
	}

	updatedMetadata, err := p.metadataService.Update(data.ID, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to update db metadata: %w", err)
	}

	msg.Data = updatedMetadata
	msg.Type = ResponseDbMetadataUpdate

	p.logger.Info().
		Uint("metadataId", data.ID).
		Uint("userId", msg.UserID).
		Msg("DB MetadataDatabase updated via WebSocket")

	return msg, nil
}

func (p *MessageProcessor) processDbNodeGuessDataModel(msg *Message) (*Message, error) {
	p.logger.Debug().Msg("Guessing data model for node")
	type nodeGuessDataModel struct {
		NodeID   int               `json:"nodeId"`
		JobID    uint              `json:"jobId"`
		Query    string            `json:"query"`
		DbType   models.DBType     `json:"dbType"`
		DbSchema string            `json:"dbSchema"`
		Host     string            `json:"host"`
		Port     int               `json:"port"`
		Database string            `json:"database"`
		Username string            `json:"username"`
		Password string            `json:"password"`
		SSLMode  string            `json:"sslMode"`
		Extra    map[string]string `json:"extra,omitempty"`
		DSN      string            `json:"dsn"`
	}
	var data nodeGuessDataModel
	if err := p.validateData(msg, &data); err != nil {
		p.logger.Error().Err(err).Msg("Failed to validate node guess data model")
		return nil, err
	}

	node := models.DBInputConfig{
		Query:    data.Query,
		DbSchema: data.DbSchema,
		Connection: models.DBConnectionConfig{
			Type:     data.DbType,
			Host:     data.Host,
			Port:     data.Port,
			Database: data.Database,
			Username: data.Username,
			Password: data.Password,
			SSLMode:  data.SSLMode,
			Extra:    data.Extra,
			DSN:      data.DSN,
		},
		DataModels: nil,
	}
	p.logger.Debug().Msgf("Node guess data: %+v", node)

	err := node.FillDataModels()
	if err != nil {
		return nil, fmt.Errorf("failed to guess data model: %w", err)
	}
	p.logger.Debug().Msgf("Guessed data models: %+v", node.DataModels)

	msg.Data = map[string]any{
		"nodeId":     data.NodeID,
		"jobId":      data.JobID,
		"dataModels": node.DataModels,
	}
	msg.Type = ResponseDbNodeGuessDataModel

	p.logger.Info().Msg("Guessed data model for node via WebSocket")
	return msg, nil

}

// SFTP MetadataDatabase CRUD operations

func (p *MessageProcessor) processSftpMetadataGet(msg *Message) (*Message, error) {
	var data struct {
		ID uint `json:"id"`
	}
	if err := p.validateData(msg, &data); err != nil {
		return nil, err
	}

	metadata, err := p.sftpMetadataService.FindByID(data.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sftp metadata: %w", err)
	}

	msg.Data = metadata
	msg.Type = ResponseSftpMetadataGet

	p.logger.Info().
		Uint("metadataId", data.ID).
		Uint("userId", msg.UserID).
		Msg("SFTP MetadataDatabase retrieved via WebSocket")

	return msg, nil
}

func (p *MessageProcessor) processSftpMetadataGetAll(msg *Message) (*Message, error) {
	metadataList, err := p.sftpMetadataService.FindAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get all sftp metadata: %w", err)
	}

	msg.Data = metadataList
	msg.Type = ResponseSftpMetadataGetAll

	p.logger.Info().
		Uint("userId", msg.UserID).
		Int("count", len(metadataList)).
		Msg("All SFTP MetadataDatabase retrieved via WebSocket")

	return msg, nil
}

func (p *MessageProcessor) processSftpMetadataCreate(msg *Message) (*Message, error) {
	var data struct {
		Host       string `json:"host"`
		Port       string `json:"port"`
		User       string `json:"user"`
		Password   string `json:"password"`
		PrivateKey string `json:"privateKey"`
		BasePath   string `json:"basePath"`
		Extra      string `json:"extra"`
	}
	if err := p.validateData(msg, &data); err != nil {
		return nil, err
	}

	metadata := models.MetadataSftp{
		Host:       data.Host,
		Port:       data.Port,
		User:       data.User,
		Password:   data.Password,
		PrivateKey: data.PrivateKey,
		BasePath:   data.BasePath,
		Extra:      data.Extra,
	}

	createdMetadata, err := p.sftpMetadataService.Create(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create sftp metadata: %w", err)
	}

	msg.Data = createdMetadata
	msg.Type = ResponseSftpMetadataCreate

	p.logger.Info().
		Uint("metadataId", createdMetadata.ID).
		Uint("userId", msg.UserID).
		Msg("SFTP MetadataDatabase created via WebSocket")

	return msg, nil
}

func (p *MessageProcessor) processSftpMetadataDelete(msg *Message) (*Message, error) {
	var data struct {
		ID uint `json:"id"`
	}
	if err := p.validateData(msg, &data); err != nil {
		return nil, err
	}

	if err := p.sftpMetadataService.Delete(data.ID); err != nil {
		return nil, fmt.Errorf("failed to delete sftp metadata: %w", err)
	}

	msg.Data = map[string]any{
		"id":      data.ID,
		"deleted": true,
	}
	msg.Type = ResponseSftpMetadataDelete

	p.logger.Info().
		Uint("metadataId", data.ID).
		Uint("userId", msg.UserID).
		Msg("SFTP MetadataDatabase deleted via WebSocket")

	return msg, nil
}

func (p *MessageProcessor) processSftpMetadataUpdate(msg *Message) (*Message, error) {
	var data struct {
		ID         uint    `json:"id"`
		Host       *string `json:"host,omitempty"`
		Port       *string `json:"port,omitempty"`
		User       *string `json:"user,omitempty"`
		Password   *string `json:"password,omitempty"`
		PrivateKey *string `json:"privateKey,omitempty"`
		BasePath   *string `json:"basePath,omitempty"`
		Extra      *string `json:"extra,omitempty"`
	}
	if err := p.validateData(msg, &data); err != nil {
		return nil, err
	}

	if data.ID == 0 {
		return nil, fmt.Errorf("sftp metadata ID is required for update")
	}

	patch := make(map[string]any)
	if data.Host != nil {
		patch["host"] = *data.Host
	}
	if data.Port != nil {
		patch["port"] = *data.Port
	}
	if data.User != nil {
		patch["user"] = *data.User
	}
	if data.Password != nil {
		patch["password"] = *data.Password
	}
	if data.PrivateKey != nil {
		patch["private_key"] = *data.PrivateKey
	}
	if data.BasePath != nil {
		patch["base_path"] = *data.BasePath
	}
	if data.Extra != nil {
		patch["extra"] = *data.Extra
	}

	updatedMetadata, err := p.sftpMetadataService.Update(data.ID, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to update sftp metadata: %w", err)
	}

	msg.Data = updatedMetadata
	msg.Type = ResponseSftpMetadataUpdate

	p.logger.Info().
		Uint("metadataId", data.ID).
		Uint("userId", msg.UserID).
		Msg("SFTP MetadataDatabase updated via WebSocket")

	return msg, nil
}
