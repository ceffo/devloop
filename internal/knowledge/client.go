// Package knowledge manages knowledge base interactions and context priming.
package knowledge

// Client defines the interface for knowledge management operations.
type Client interface {
	// Prime returns priming context to inject into prompts
	Prime() string

	// Record stores knowledge from completed tasks or interactions
	Record(data any) error

	// IsEnabled returns whether knowledge management is active
	IsEnabled() bool
}

// NewClient creates a new knowledge client based on the provided backend configuration.
// Returns a NoopClient when backend is empty or 'none'.
func NewClient(backend string) Client {
	if backend == "" || backend == "none" {
		return &NoopClient{}
	}
	// For now, only noop client is implemented
	return &NoopClient{}
}
