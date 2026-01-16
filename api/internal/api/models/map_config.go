package models

// FunctionType defines the type of transformation function
type FunctionType string

const (
	FuncTypeDirect  FunctionType = "direct"  // Direct column mapping: out.col = input.col
	FuncTypeLibrary FunctionType = "library" // Standard library function: lib.Concat(...)
	FuncTypeCustom  FunctionType = "custom"  // User-defined expression or function
)

// CustomFuncType defines whether custom is an expression or full function
type CustomFuncType string

const (
	CustomExpr CustomFuncType = "expr" // Single expression: A.price * 1.2
	CustomFunc CustomFuncType = "func" // Full function declaration
)

// JoinType defines how multiple inputs are combined
type JoinType string

const (
	JoinTypeInner JoinType = "inner"
	JoinTypeLeft  JoinType = "left"
	JoinTypeRight JoinType = "right"
	JoinTypeFull  JoinType = "full"
	JoinTypeCross JoinType = "cross"
	JoinTypeUnion JoinType = "union" // Concatenate rows (no key matching)
)

// InputFlow represents an input data stream with its schema
type InputFlow struct {
	Name   string      `json:"name"`   // Reference name (e.g., "A", "orders")
	PortID int         `json:"portId"` // Connected input port ID
	Schema []DataModel `json:"schema"` // Column definitions from upstream
}

// OutputFlow represents an output data stream with its own port
type OutputFlow struct {
	Name    string         `json:"name"`    // Output flow name
	PortID  int            `json:"portId"`  // Output port ID (each output has its own)
	Columns []MapOutputCol `json:"columns"` // Column definitions for this output
}

// MapOutputCol defines how an output column is computed
type MapOutputCol struct {
	Name     string       `json:"name"`     // Output column name
	DataType string       `json:"dataType"` // Output data type (string, int, float64, bool, time.Time)
	FuncType FunctionType `json:"funcType"` // Type of transformation

	// For FuncTypeDirect - direct column reference
	InputRef string `json:"inputRef,omitempty"` // "A.column_name" or "orders.id"

	// For FuncTypeLibrary - standard library function
	LibFunc string    `json:"libFunc,omitempty"` // Function name: "Concat", "Upper", "Add", etc.
	Args    []FuncArg `json:"args,omitempty"`    // Function arguments

	// For FuncTypeCustom - user-defined transformation
	CustomType CustomFuncType `json:"customType,omitempty"` // "expr" or "func"
	Expression string         `json:"expression,omitempty"` // For expr: "A.price * 1.2"
	FuncBody   string         `json:"funcBody,omitempty"`   // For func: full function body
}

// FuncArg represents a function argument
type FuncArg struct {
	Type  string `json:"type"`  // "column", "literal"
	Value string `json:"value"` // Column ref ("A.name") or literal value ("hello", "100")
}

// JoinConfig defines how multiple inputs are combined
type JoinConfig struct {
	Type       JoinType `json:"type"`                 // Join type
	LeftInput  string   `json:"leftInput"`            // Left input name
	RightInput string   `json:"rightInput"`           // Right input name
	LeftKey    string   `json:"leftKey,omitempty"`    // Left join column (not needed for union/cross)
	RightKey   string   `json:"rightKey,omitempty"`   // Right join column
	LeftKeys   []string `json:"leftKeys,omitempty"`   // For composite keys
	RightKeys  []string `json:"rightKeys,omitempty"`  // For composite keys
}

// MapConfig is the complete configuration for a Map node
type MapConfig struct {
	Inputs  []InputFlow  `json:"inputs"`          // Input streams (1 or more)
	Outputs []OutputFlow `json:"outputs"`         // Output streams (1 or more, each with own port)
	Join    *JoinConfig  `json:"join,omitempty"`  // How to combine multiple inputs (nil if single input)
}

// GetInputByName returns an input flow by its reference name
func (c *MapConfig) GetInputByName(name string) *InputFlow {
	for i := range c.Inputs {
		if c.Inputs[i].Name == name {
			return &c.Inputs[i]
		}
	}
	return nil
}

// GetOutputByName returns an output flow by its name
func (c *MapConfig) GetOutputByName(name string) *OutputFlow {
	for i := range c.Outputs {
		if c.Outputs[i].Name == name {
			return &c.Outputs[i]
		}
	}
	return nil
}

// GetOutputByPortID returns an output flow by its port ID
func (c *MapConfig) GetOutputByPortID(portID int) *OutputFlow {
	for i := range c.Outputs {
		if c.Outputs[i].PortID == portID {
			return &c.Outputs[i]
		}
	}
	return nil
}

// HasMultipleInputs returns true if the map node has more than one input
func (c *MapConfig) HasMultipleInputs() bool {
	return len(c.Inputs) > 1
}

// HasMultipleOutputs returns true if the map node has more than one output
func (c *MapConfig) HasMultipleOutputs() bool {
	return len(c.Outputs) > 1
}