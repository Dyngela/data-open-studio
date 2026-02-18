package mapper

import (
	"api/internal/api/handler/request"
	"api/internal/api/handler/response"
	"api/internal/api/models"
)

// DatasetMapper handles mapping between dataset models and DTOs
type DatasetMapper interface {
	ToDataset(req request.CreateDataset, creatorID uint) models.Dataset
	ToDatasetPatch(req request.UpdateDataset) models.Dataset
	ToDatasetSummary(d models.Dataset) response.DatasetSummary
	ToDatasetSummaries(datasets []models.Dataset) []response.DatasetSummary
	ToDatasetWithDetails(d models.Dataset) response.DatasetWithDetails
	ToQueryFilters(req []request.DatasetQueryFilter) []models.QueryFilter
}

type DatasetMapperImpl struct{}

func NewDatasetMapper() DatasetMapper {
	return &DatasetMapperImpl{}
}

func (m *DatasetMapperImpl) ToDataset(req request.CreateDataset, creatorID uint) models.Dataset {
	return models.Dataset{
		Name:               req.Name,
		Description:        req.Description,
		CreatorID:          creatorID,
		MetadataDatabaseID: req.MetadataDatabaseID,
		Query:              req.Query,
	}
}

func (m *DatasetMapperImpl) ToDatasetPatch(req request.UpdateDataset) models.Dataset {
	patch := models.Dataset{}
	if req.Name != nil {
		patch.Name = *req.Name
	}
	if req.Description != nil {
		patch.Description = *req.Description
	}
	if req.MetadataDatabaseID != nil {
		patch.MetadataDatabaseID = *req.MetadataDatabaseID
	}
	if req.Query != nil {
		patch.Query = *req.Query
	}
	return patch
}

func (m *DatasetMapperImpl) ToDatasetSummary(d models.Dataset) response.DatasetSummary {
	return response.DatasetSummary{
		ID:                 d.ID,
		Name:               d.Name,
		Description:        d.Description,
		CreatorID:          d.CreatorID,
		MetadataDatabaseID: d.MetadataDatabaseID,
		Status:             d.Status,
		ColumnCount:        len(d.Schema.Columns),
		LastRefreshedAt:    d.LastRefreshedAt,
		LastError:          d.LastError,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

func (m *DatasetMapperImpl) ToDatasetSummaries(datasets []models.Dataset) []response.DatasetSummary {
	result := make([]response.DatasetSummary, len(datasets))
	for i, d := range datasets {
		result[i] = m.ToDatasetSummary(d)
	}
	return result
}

func (m *DatasetMapperImpl) ToDatasetWithDetails(d models.Dataset) response.DatasetWithDetails {
	return response.DatasetWithDetails{
		ID:                 d.ID,
		Name:               d.Name,
		Description:        d.Description,
		CreatorID:          d.CreatorID,
		MetadataDatabaseID: d.MetadataDatabaseID,
		Query:              d.Query,
		Schema:             d.Schema,
		Status:             d.Status,
		LastRefreshedAt:    d.LastRefreshedAt,
		LastError:          d.LastError,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

func (m *DatasetMapperImpl) ToQueryFilters(req []request.DatasetQueryFilter) []models.QueryFilter {
	filters := make([]models.QueryFilter, len(req))
	for i, f := range req {
		filters[i] = models.QueryFilter{
			Column:   f.Column,
			Operator: f.Operator,
			Value:    f.Value,
		}
	}
	return filters
}
