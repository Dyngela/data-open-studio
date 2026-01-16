package websocket

import (
	"api/internal/api/models"
	"api/internal/api/service"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
)

// MessageProcessor handles WebSocket messages and performs database operations
type MessageProcessor struct {
	jobService *service.JobService
	logger     zerolog.Logger
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(jobService *service.JobService, logger zerolog.Logger) *MessageProcessor {
	return &MessageProcessor{
		jobService: jobService,
		logger:     logger,
	}
}

// ProcessMessage processes a message and performs necessary database operations
// Returns the updated message to broadcast, or error if processing failed
func (p *MessageProcessor) ProcessMessage(msg *Message) (*Message, error) {
	switch msg.Type {
	case MessageTypeJobUpdate:
		return p.processJobUpdate(msg)
	case MessageTypeJobDelete:
		return p.processJobDelete(msg)
	case MessageTypeJobCreate:
		return p.processJobCreate(msg)
	case MessageTypeJobExecute:
		return p.processJobExecute(msg)

	case MessageTypeMetadataGet:
		return p.processMetadataGet(msg)
	case MessageTypeMetadataUpdate:
		return p.processMetadataUpdate(msg)
	case MessageTypeMetadataDelete:
		return p.processMetadataDelete(msg)
	case MessageTypeMetadataCreate:
		return p.processMetadataCreate(msg)

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
		// Convert nodes if provided
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
func (p *MessageProcessor) processMetadataGet(msg *Message) (*Message, error) {
	return nil, nil
}
func (p *MessageProcessor) processMetadataCreate(msg *Message) (*Message, error) {
	return nil, nil

}
func (p *MessageProcessor) processMetadataDelete(msg *Message) (*Message, error) {
	return nil, nil

}
func (p *MessageProcessor) processMetadataUpdate(msg *Message) (*Message, error) {
	return nil, nil

}

func (p *MessageProcessor) processDbNodeGuessDataModel(msg *Message) (*Message, error) {
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

	err := node.FillDataModels()
	if err != nil {
		return nil, fmt.Errorf("failed to guess data model: %w", err)
	}

	msg.Data = map[string]any{
		"nodeId":     data.NodeID,
		"jobId":      data.JobID,
		"dataModels": node.DataModels,
	}
	msg.Type = ResponseDbNodeGuessDataModel

	return msg, nil
}
