package mapper

import (
	"api/internal/api/handler/request"
	"api/internal/api/handler/response"
	"api/internal/api/models"
)

// TriggerMapper handles mapping between trigger models and DTOs
type TriggerMapper interface {
	CreateTrigger(req request.CreateTrigger, creatorID uint) models.Trigger
	PatchTrigger(req request.UpdateTrigger) map[string]interface{}
	ToTriggerResponse(t models.Trigger) response.Trigger
	ToTriggerResponses(triggers []models.Trigger) []response.Trigger
	ToTriggerWithDetails(t models.Trigger) response.TriggerWithDetails
	ToTriggerRuleResponse(r models.TriggerRule) response.TriggerRule
	ToTriggerExecutionResponse(e models.TriggerExecution) response.TriggerExecution
	ToTriggerExecutionResponses(execs []models.TriggerExecution) []response.TriggerExecution
}

// TriggerMapperImpl implements TriggerMapper
type TriggerMapperImpl struct{}

// NewTriggerMapper creates a new TriggerMapper instance
func NewTriggerMapper() TriggerMapper {
	return &TriggerMapperImpl{}
}

// CreateTrigger maps a create request to a trigger model
func (m *TriggerMapperImpl) CreateTrigger(req request.CreateTrigger, creatorID uint) models.Trigger {
	trigger := models.Trigger{
		Name:            req.Name,
		Description:     req.Description,
		Type:            req.Type,
		Status:          models.TriggerStatusPaused,
		CreatorID:       creatorID,
		PollingInterval: req.PollingInterval,
		Config:          req.Config,
	}
	if trigger.PollingInterval == 0 {
		trigger.PollingInterval = 60 // Default 1 minute
	}
	return trigger
}

// PatchTrigger maps an update request to a patch map
func (m *TriggerMapperImpl) PatchTrigger(req request.UpdateTrigger) map[string]interface{} {
	patch := make(map[string]interface{})
	if req.Name != nil {
		patch["name"] = *req.Name
	}
	if req.Description != nil {
		patch["description"] = *req.Description
	}
	if req.PollingInterval != nil {
		patch["polling_interval"] = *req.PollingInterval
	}
	if req.Config != nil {
		patch["config"] = *req.Config
	}
	return patch
}

// ToTriggerResponse maps a trigger model to a response (list view)
func (m *TriggerMapperImpl) ToTriggerResponse(t models.Trigger) response.Trigger {
	return response.Trigger{
		ID:              t.ID,
		Name:            t.Name,
		Description:     t.Description,
		Type:            t.Type,
		Status:          t.Status,
		CreatorID:       t.CreatorID,
		PollingInterval: t.PollingInterval,
		LastPolledAt:    t.LastPolledAt,
		LastError:       t.LastError,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
		JobCount:        len(t.Jobs),
	}
}

// ToTriggerResponses maps a slice of trigger models to responses
func (m *TriggerMapperImpl) ToTriggerResponses(triggers []models.Trigger) []response.Trigger {
	result := make([]response.Trigger, len(triggers))
	for i, t := range triggers {
		result[i] = m.ToTriggerResponse(t)
	}
	return result
}

// ToTriggerWithDetails maps a trigger model to a detailed response
func (m *TriggerMapperImpl) ToTriggerWithDetails(t models.Trigger) response.TriggerWithDetails {
	resp := response.TriggerWithDetails{
		ID:              t.ID,
		Name:            t.Name,
		Description:     t.Description,
		Type:            t.Type,
		Status:          t.Status,
		CreatorID:       t.CreatorID,
		PollingInterval: t.PollingInterval,
		LastPolledAt:    t.LastPolledAt,
		LastError:       t.LastError,
		Config:          t.Config,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
		Rules:           make([]response.TriggerRule, len(t.Rules)),
		Jobs:            make([]response.TriggerJobLink, len(t.Jobs)),
	}

	for i, r := range t.Rules {
		resp.Rules[i] = m.ToTriggerRuleResponse(r)
	}

	for i, j := range t.Jobs {
		resp.Jobs[i] = response.TriggerJobLink{
			ID:            j.ID,
			TriggerID:     j.TriggerID,
			JobID:         j.JobID,
			JobName:       j.Job.Name,
			Priority:      j.Priority,
			Active:        j.Active,
			PassEventData: j.PassEventData,
		}
	}

	return resp
}

// ToTriggerRuleResponse maps a rule model to a response
func (m *TriggerMapperImpl) ToTriggerRuleResponse(r models.TriggerRule) response.TriggerRule {
	return response.TriggerRule{
		ID:         r.ID,
		TriggerID:  r.TriggerID,
		Name:       r.Name,
		Conditions: r.Conditions,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

// ToTriggerExecutionResponse maps an execution model to a response
func (m *TriggerMapperImpl) ToTriggerExecutionResponse(e models.TriggerExecution) response.TriggerExecution {
	return response.TriggerExecution{
		ID:            e.ID,
		TriggerID:     e.TriggerID,
		StartedAt:     e.StartedAt,
		FinishedAt:    e.FinishedAt,
		Status:        e.Status,
		EventCount:    e.EventCount,
		JobsTriggered: e.JobsTriggered,
		Error:         e.Error,
		EventSample:   e.EventSample,
	}
}

// ToTriggerExecutionResponses maps a slice of execution models to responses
func (m *TriggerMapperImpl) ToTriggerExecutionResponses(execs []models.TriggerExecution) []response.TriggerExecution {
	result := make([]response.TriggerExecution, len(execs))
	for i, e := range execs {
		result[i] = m.ToTriggerExecutionResponse(e)
	}
	return result
}
