// Package httputil provides shared HTTP response types for Swagger documentation.
package httputil

// ErrorResponse is the generic fallback standard error envelope string.
type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error"   example:"something went wrong"`
}

// Error400Response is returned for bad requests or validation errors.
type Error400Response struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error"   example:"invalid request body"`
}

// Error401Response is returned when the user is not authenticated.
type Error401Response struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error"   example:"unauthorized"`
}

// Error403Response is returned when the user lacks required permissions.
type Error403Response struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error"   example:"forbidden accesss"`
}

// Error404Response is returned when a requested resource is not found.
type Error404Response struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error"   example:"resource not found"`
}

// Error500Response is returned for internal server errors.
type Error500Response struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error"   example:"internal server error"`
}

// MessageResponse is returned for operations that produce only a string message.
type MessageResponse struct {
	Success bool   `json:"success" example:"true"`
	Data    string `json:"data"    example:"ok"`
}
