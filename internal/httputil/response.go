// Package httputil provides shared HTTP response types for Swagger documentation.
package httputil

// ErrorResponse is the standard error envelope returned on failure.
type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error"   example:"something went wrong"`
}

// MessageResponse is returned for operations that produce only a string message.
type MessageResponse struct {
	Success bool   `json:"success" example:"true"`
	Data    string `json:"data"    example:"ok"`
}
