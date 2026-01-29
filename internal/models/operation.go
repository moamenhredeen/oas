package models

// Operation represents an OpenAPI operation with test context
type Operation struct {
	Path        string
	Method      string
	OperationID string
	Tags        []string
	ServerURL   string
	FullPath    string // ServerURL + Path with parameters resolved
}
