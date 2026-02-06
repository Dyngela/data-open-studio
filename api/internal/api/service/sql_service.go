package service

import (
	"api"
	"api/internal/api/models"
	"api/pkg"
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const cacheTTL = 60 * time.Minute

type SqlService struct {
	logger          zerolog.Logger
	metadataService *MetadataService
}

func NewSqlService() *SqlService {
	return &SqlService{
		logger:          api.Logger,
		metadataService: NewMetadataService(),
	}
}

func (slf *SqlService) GuessQuery(prompt string, schemaOptimizationNeeded bool, connectionID int, previousMessages []string) (string, error) {
	// 1. Résoudre la metadata de connexion -> cache ou DB
	metadata, err := slf.resolveMetadata(connectionID)
	if err != nil {
		return "", err
	}

	// 2. Trouver les business rules -> cache (pas de modèle DB pour l'instant, on stocke un []string vide par défaut)
	businessRules, err := slf.resolveBusinessRules(connectionID)
	if err != nil {
		return "", err
	}

	// 3. Trouver le schéma de la DB -> cache ou requête directe
	dbSchema, err := slf.resolveSchema(connectionID, *metadata)
	if err != nil {
		return "", err
	}

	// 4. Optimiser le schéma si nécessaire (ne garder que les tables pertinentes)
	if schemaOptimizationNeeded {
		relevant, err := pkg.GuessRelevantTablesWithInput(nil, prompt, dbSchema, businessRules, metadata.DbType)
		if err != nil {
			slf.logger.Warn().Err(err).Msg("schema optimization failed, using full schema")
		} else if len(relevant) > 0 {
			dbSchema = relevant
		}
	}

	// 5. Déterminer si on doit inclure les métadonnées dans le prompt
	// On les inclut sur la première requête OU quand on dépasse la limite de messages en cache
	messageLimit := api.GetConfig().OllamaMessageLimit
	includeMetadata := len(previousMessages) == 0 || len(previousMessages) >= messageLimit

	// 6. Appeler le LLM pour générer la requête
	resp, err := pkg.GuessQuery(nil, prompt, dbSchema, businessRules, previousMessages, metadata.DbType, includeMetadata)
	if err != nil {
		return "", fmt.Errorf("guess query failed: %w", err)
	}

	return resp.Query, nil
}

func (slf *SqlService) OptimizeQuery(query string, connectionId int) (string, string, error) {
	if !pkg.IsSafeSelect(query) {
		return "", "", fmt.Errorf("query is not a select query, can't optimize that: %s", query)
	}

	metadata, err := slf.resolveMetadata(connectionId)
	if err != nil {
		return "", "", err
	}

	resp, err := pkg.OptimizeQuery(nil, query, metadata.DbType)
	if err != nil {
		return "", "", err
	}
	return resp.OptimizedQuery, resp.Explanation, nil
}

func (slf *SqlService) resolveMetadata(connectionID int) (*models.MetadataDatabase, error) {
	cacheKey := fmt.Sprintf("conn:%d:meta", connectionID)
	var metadata models.MetadataDatabase
	if err := pkg.RedisGet(cacheKey, &metadata); err != nil {
		if !pkg.IsRedisNil(err) {
			return nil, fmt.Errorf("redis error: %w", err)
		}
		found, err := slf.metadataService.FindByID(uint(connectionID))
		if err != nil {
			return nil, fmt.Errorf("connection %d not found: %w", connectionID, err)
		}
		metadata = *found
		_ = pkg.RedisSet(cacheKey, metadata, cacheTTL)
	}
	return &metadata, nil
}

func (slf *SqlService) resolveBusinessRules(connectionID int) ([]string, error) {
	cacheKey := fmt.Sprintf("conn:%d:rules", connectionID)
	var rules []string
	if err := pkg.RedisGet(cacheKey, &rules); err != nil {
		if !pkg.IsRedisNil(err) {
			return nil, fmt.Errorf("redis error: %w", err)
		}
		rules = []string{}
		_ = pkg.RedisSet(cacheKey, rules, cacheTTL)
	}
	return rules, nil
}

func (slf *SqlService) resolveSchema(connectionID int, metadata models.MetadataDatabase) ([]pkg.TableMetadata, error) {
	cacheKey := fmt.Sprintf("conn:%d:schema", connectionID)
	var schema []pkg.TableMetadata
	if err := pkg.RedisGet(cacheKey, &schema); err != nil {
		if !pkg.IsRedisNil(err) {
			return nil, fmt.Errorf("redis error: %w", err)
		}
		fetched, err := slf.fetchSchema(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch schema: %w", err)
		}
		schema = fetched
		_ = pkg.RedisSet(cacheKey, schema, cacheTTL)
	}
	return schema, nil
}

func (slf *SqlService) fetchSchema(metadata models.MetadataDatabase) ([]pkg.TableMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch metadata.DbType {
	case models.DBTypePostgres:
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			metadata.Host, metadata.Port, metadata.User, metadata.Password, metadata.DatabaseName, metadata.SSLMode)
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return nil, err
		}
		defer pool.Close()
		return pkg.FindPostgresSchemaDatabaseSchema(ctx, pool)

	case models.DBTypeSQLServer:
		connCfg := models.DBConnectionConfig{
			Type:     models.DBTypeSQLServer,
			Host:     metadata.Host,
			Port:     metadata.Port,
			Username: metadata.User,
			Password: metadata.Password,
			Database: metadata.DatabaseName,
			SSLMode:  metadata.SSLMode,
		}
		db, err := sql.Open("sqlserver", connCfg.BuildConnectionString())
		if err != nil {
			return nil, err
		}
		defer db.Close()
		return pkg.FindSQLServerSchemaDatabaseSchema(ctx, db)

	default:
		return nil, fmt.Errorf("unsupported database type for schema fetch: %s", metadata.DbType)
	}
}
