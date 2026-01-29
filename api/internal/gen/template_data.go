package gen

// TemplateData holds all data needed to generate the main.go file
type TemplateData struct {
	Imports       []ImportData
	Structs       []StructData
	NodeFunctions []NodeFunctionData
	DBConnections []DBConnectionData
	Channels      []ChannelData
	NodeLaunches  []NodeLaunchData
	NodeCount     int

	// Progress reporting config
	UseFlags bool
	NatsURL  string
	TenantID string
	JobID    uint
}

// ImportData represents an import statement
type ImportData struct {
	Path  string
	Alias string // empty for no alias
}

// StructData represents a struct definition
type StructData struct {
	Name   string
	NodeID int
	Fields []FieldData
}

// FieldData represents a struct field
type FieldData struct {
	Name string
	Type string
	Tag  string
}

// NodeFunctionData represents a node's execution function
type NodeFunctionData struct {
	Name      string
	NodeID    int
	NodeName  string
	Signature string
	Body      string
}

// DBConnectionData represents a database connection
type DBConnectionData struct {
	ID         string
	Driver     string
	ConnString string
}

// ChannelData represents a channel between nodes
type ChannelData struct {
	PortID     uint
	FromNodeID int
	ToNodeID   int
	RowType    string
	BufferSize int
}

// NodeLaunchData represents goroutine launch data for a node
type NodeLaunchData struct {
	NodeID           int
	NodeName         string
	FuncName         string
	Args             []string
	HasOutputChannel bool
	OutputChannel    string
}

// MapTransformTemplateData holds data for map transformation template
type MapTransformTemplateData struct {
	FuncName   string
	NodeID     int
	NodeName   string
	InputType  string
	OutputType string
	Transforms string
}

// MapJoinTemplateData holds data for map join templates
type MapJoinTemplateData struct {
	FuncName   string
	NodeID     int
	NodeName   string
	LeftType   string
	RightType  string
	OutputType string
	LeftKey    string
	RightKey   string
	Transforms string
}

// MapUnionTemplateData holds data for map union template
type MapUnionTemplateData struct {
	FuncName        string
	NodeID          int
	NodeName        string
	LeftType        string
	RightType       string
	OutputType      string
	LeftTransforms  string
	RightTransforms string
}

// LogTemplateData holds data for log node template
type LogTemplateData struct {
	FuncName  string
	NodeID    int
	NodeName  string
	InputType string
}

// DBOutputInsertTemplateData holds data for db_output INSERT template
type DBOutputInsertTemplateData struct {
	FuncName       string
	NodeID         int
	NodeName       string
	InputType      string
	TableName      string
	ColumnNames    string
	NumColumns     int
	FieldAccessors []string
	BatchSize      int
}
