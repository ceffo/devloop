package knowledge

// NoopClient is a no-operation implementation of Client that is used when
// knowledge management is disabled.
type NoopClient struct{}

// Prime returns an empty string
func (n *NoopClient) Prime() string {
	return ""
}

// Record returns nil (no-op)
func (n *NoopClient) Record(data any) error {
	return nil
}

// IsEnabled returns false
func (n *NoopClient) IsEnabled() bool {
	return false
}
