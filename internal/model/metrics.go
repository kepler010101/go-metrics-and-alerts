// Package models contains shared data structures between server and agent.
package models

const (
	// Counter marks counter metrics.
	Counter = "counter"
	// Gauge marks gauge metrics.
	Gauge = "gauge"
)

// Metrics encodes a metric payload shared by the agent and the server.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}
