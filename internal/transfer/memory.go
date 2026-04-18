package transfer

// Memory is the CLI-local shape used by import/export parsers and formatters.
// It mirrors the subset of server-side memory fields the wire formats need:
// content, type, namespace, metadata, and — for exports — id/created_at.
// Defined here (rather than imported from the main repo's internal/memory
// package) so the CLI can stand alone as a public module.
type Memory struct {
	ID        string  `json:"id,omitempty"`
	Namespace string  `json:"namespace"`
	Type      string  `json:"type"`
	Content   string  `json:"content"`
	Metadata  *string `json:"metadata,omitempty"`
	SourceID  *string `json:"source_id,omitempty"`
	Version   int     `json:"version,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
	UpdatedAt string  `json:"updated_at,omitempty"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}
