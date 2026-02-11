package models

type NodeLogConfig struct {
	Input     []DataModel `json:"input,omitempty"`
	Separator string      `json:"separator,omitempty"`
}
