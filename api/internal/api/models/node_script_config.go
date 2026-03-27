package models

// ScriptConfig holds configuration for a script node
type ScriptConfig struct {
	Inputs     []ScriptInputSelection `json:"inputs"`
	DataModels []DataModel            `json:"dataModels"`
	Code       string                 `json:"code"`
}

// ScriptInputSelection maps a named variable to an input port
type ScriptInputSelection struct {
	Name   string `json:"name"`
	PortID uint   `json:"portId"`
}
