package models

// DBInputGenerator handles database input operations
type DBInputGenerator struct {
	BaseGenerator
	config DBInputConfig
}

// NewDBInputGenerator creates a new DB input generator
func NewDBInputGenerator(nodeID int, config DBInputConfig) *DBInputGenerator {
	return &DBInputGenerator{
		BaseGenerator: BaseGenerator{
			nodeID:   nodeID,
			nodeType: NodeTypeDBInput,
		},
		config: config,
	}
}

// Execute reads data from database and stores in context
func (g *DBInputGenerator) Execute(ctx *ExecutionContext) error {

	return nil
}
