package pkg

import (
	"api"
	"api/internal/api/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

type ollamaRawResponse struct {
	Model   string `json:"model"`
	Message struct {
		Content string `json:"content"`
	} `json:"message"` // C'est ici que se trouve votre JSON structuré
	Done bool `json:"done"`
}

type ollamaApiCall struct {
	Model    string         `json:"model"`
	Prompt   string         `json:"prompt"`
	Messages []string       `json:"messages,omitempty"`
	Stream   bool           `json:"stream"`
	Format   map[string]any `json:"format"`
	Options  map[string]any `json:"options"`
}

func (slf *ollamaApiCall) new(model string, schema map[string]any, prompt string, messages []string) *ollamaApiCall {

	return &ollamaApiCall{
		Model:    model,
		Prompt:   prompt,
		Messages: messages,
		Stream:   false,
		Format:   schema,
		Options: map[string]any{
			"temperature": 0,
		},
	}
}

func (slf *ollamaApiCall) callGuessQuery() (OllamaGuessQueryResponse, error) {
	var result OllamaGuessQueryResponse

	host := api.GetConfig().OllamaHost

	data, err := json.Marshal(slf)
	if err != nil {
		AssertNoError(err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/chat", host),
		bytes.NewBuffer(data),
	)
	if err != nil {
		AssertNoError(err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		AssertNoError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var raw ollamaRawResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return result, err
	}

	if !raw.Done {
		return result, fmt.Errorf("llama call not done")
	}

	// On parse le contenu du champ 'response' qui contient le JSON généré par l'IA [3]
	if err := json.Unmarshal([]byte(raw.Message.Content), &result); err != nil {
		return result, err
	}

	return result, nil

}

func (slf *ollamaApiCall) callGuessSchemaRelevance() (OllamaGuessSchemaRelevanceResponse, error) {
	var result OllamaGuessSchemaRelevanceResponse

	data, err := json.Marshal(slf)
	if err != nil {
		AssertNoError(err)
	}
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/chat", api.GetConfig().OllamaHost),
		bytes.NewBuffer(data),
	)
	if err != nil {
		AssertNoError(err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		AssertNoError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var raw ollamaRawResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return result, err
	}

	if !raw.Done {
		return result, fmt.Errorf("llama call not done")
	}

	// On parse le contenu du champ 'response' qui contient le JSON généré par l'IA [3]
	if err := json.Unmarshal([]byte(raw.Message.Content), &result); err != nil {
		return result, err
	}

	return result, nil
}

func (slf *ollamaApiCall) callOptimizeQuery() (OllamaOptimizeQueryResponse, error) {
	var result OllamaOptimizeQueryResponse

	data, err := json.Marshal(slf)
	if err != nil {
		AssertNoError(err)
	}
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/chat", api.GetConfig().OllamaHost),
		bytes.NewBuffer(data),
	)
	if err != nil {
		AssertNoError(err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		AssertNoError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	var raw ollamaRawResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return result, err
	}

	if !raw.Done {
		return result, fmt.Errorf("llama call not done")
	}

	if err := json.Unmarshal([]byte(raw.Message.Content), &result); err != nil {
		return result, err
	}

	return result, nil
}

type OllamaGuessQueryResponse struct {
	Query       string `json:"query"`
	Explanation string `json:"explanation"`
}

type OllamaGuessSchemaRelevanceResponse struct {
	RelevantTables []string `json:"relevant_tables"`
}

type OllamaOptimizeQueryResponse struct {
	OptimizedQuery string `json:"optimized_query"`
	Explanation    string `json:"explanation"`
}

// GuessQuery
/* Guesses a query based on the given prompt. Use Ollama under the hood.
model: the data model of the database. Can be nil. If so, a default model will be used.
userInput: the user's input query
dbSchema: the database schema with columns and table descriptions
businessRules: a list of business rules to apply to the query
chatContext: the chat context to use for the query in case of subsequents request */
func GuessQuery(model *string, userInput string, dbSchema []TableMetadata, businessRules []string, chatContext []string, dbType models.DBType, includeMetadata bool) (OllamaGuessQueryResponse, error) {
	if model == nil {
		model = ToPtr("qwen3-coder:30b")
	}

	// build prompt with business rules and dbSchema and userInput if needed
	var metadata string
	if includeMetadata {
		metadata = fmt.Sprintf(`
### SCHÉMA DES TABLES (%s) 
%s

### RÈGLES MÉTIER SÉMANTIQUES :
%s`, dbType, TableMetadataToLLMFormat(dbSchema), strings.Join(businessRules, "\n"))
	}

	var prompt string
	prompt = fmt.Sprintf(`Tu es un expert PostgreSQL spécialisé en analyse marketing. Ton but est de générer une requête SQL SELECT performante basée sur le schéma fourni ci-dessous. 
%s
### QUESTION UTILISATEUR: 
%s

`, metadata, userInput)

	// Pour éviter des réponse IA trop longue on ne prend que les N derniers messages
	var messageLimit = api.GetConfig().OllamaMessageLimit
	if len(chatContext) > messageLimit {
		chatContext = chatContext[len(chatContext)-messageLimit:]
	}
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":       map[string]any{"type": "string"},
			"explanation": map[string]any{"type": "string"},
		},
		"required": []string{"query", "explanation"},
	}
	ollamaCall := (&ollamaApiCall{}).new(*model, schema, prompt, chatContext)

	resp, err := ollamaCall.callGuessQuery()
	if err != nil {
		return OllamaGuessQueryResponse{}, err
	}

	if !IsSafeSelect(resp.Query) {
		return OllamaGuessQueryResponse{}, fmt.Errorf("optimized query is not a select query: %s", resp.Query)
	}

	return resp, nil
}

// GuessRelevantTablesWithInput guesses the relevant tables for a given user input, excluding of the context any other one to prevent flooding the model with irrelevant information.
func GuessRelevantTablesWithInput(model *string, userInput string, dbSchema []TableMetadata, businessRules []string, dbType models.DBType) ([]TableMetadata, error) {
	if model == nil {
		model = ToPtr("qwen3-coder:30b")
	}

	prompt := fmt.Sprintf(`Tu es un expert en analyse de schéma de base de données. Ton but est d'identifier les tables les plus pertinentes pour répondre à la question de l'utilisateur, en te basant sur le schéma fourni ci-dessous.
### SCHÉMA DES TABLES (%s) 
%s

### RÈGLES MÉTIER SÉMANTIQUES :
%s

### QUESTION UTILISATEUR: 
%s

### INSTRUCTION:
Identifie les tables les plus pertinentes pour répondre à la question de l'utilisateur. Ne réponds que par une liste de noms de tables séparés par des virgules, sans aucune explication supplémentaire.

`, dbType, TableMetadataToLLMFormat(dbSchema), strings.Join(businessRules, "\n"), userInput)

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"relevant_tables": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []string{"relevant_tables"},
	}

	ollamaCall := (&ollamaApiCall{}).new(*model, schema, prompt, nil)
	resp, err := ollamaCall.callGuessSchemaRelevance()
	if err != nil {
		return nil, err
	}

	// on trouve les tables les plus pertinentes
	var relevantTables []TableMetadata
	for _, table := range dbSchema {
		if slices.Contains(resp.RelevantTables, table.TableName) {
			relevantTables = append(relevantTables, table)
		}
	}

	return relevantTables, nil

}

// OptimizeQuery
/* Optimizes an SQL query as much as possible based on the database schema and business rules. Uses Ollama under the hood. **USE FOR SELECT ONLY**
model: the LLM model to use. Can be nil. If so, a default model will be used.
sqlQuery: the SQL query to optimize
dbSchema: the database schema with columns and table descriptions
businessRules: a list of business rules to consider during optimization
dbType: the type of database (PostgreSQL, SQL Server, etc.) */
func OptimizeQuery(model *string, sqlQuery string, dbType models.DBType) (OllamaOptimizeQueryResponse, error) {
	if model == nil {
		model = ToPtr("qwen3-coder:30b")
	}

	prompt := fmt.Sprintf(`Tu es un expert en optimisation de requêtes SQL. Ton but est d'optimiser au maximum la requête SQL fournie en te basant sur le schéma de la base de données et les règles métier.

### REQUÊTE SQL À OPTIMISER (%s) :
%s

### INSTRUCTIONS D'OPTIMISATION :
- Utilise des index-friendly patterns (évite les fonctions sur les colonnes dans les WHERE)
- Remplace les sous-requêtes corrélées par des JOIN ou des CTE quand c'est plus performant
- Élimine les colonnes inutiles dans le SELECT (pas de SELECT * si non nécessaire)
- Utilise EXISTS au lieu de IN pour les sous-requêtes quand applicable
- Optimise les JOIN (ordre, type, conditions)
- Ajoute des filtres le plus tôt possible pour réduire les ensembles de données
- Évite les opérations redondantes ou les calculs dupliqués
- La requête optimisée DOIT retourner exactement le même résultat que l'originale

`, dbType, sqlQuery)

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"optimized_query": map[string]any{"type": "string"},
			"explanation":     map[string]any{"type": "string"},
		},
		"required": []string{"optimized_query", "explanation"},
	}

	ollamaCall := (&ollamaApiCall{}).new(*model, schema, prompt, nil)

	resp, err := ollamaCall.callOptimizeQuery()
	if err != nil {
		return OllamaOptimizeQueryResponse{}, err
	}

	if !IsSafeSelect(resp.OptimizedQuery) {
		return OllamaOptimizeQueryResponse{}, fmt.Errorf("optimized query is not a select query: %s", resp.OptimizedQuery)
	}

	return resp, nil
}
