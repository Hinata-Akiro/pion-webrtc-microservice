package utils

import (
	"crypto/rand"
	"fmt"
	"time"
)




// ErrorResponse represents a structured error response
type ErrorResponse struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

// NewErrorResponse creates a new ErrorResponse
func NewErrorResponse(statusCode int, message string) *ErrorResponse {
	return &ErrorResponse{
		StatusCode: statusCode,
		Message:    message,
	}
}

// SuccessResponse represents a structured success response
type SuccessResponse struct {
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"` 
}

// NewSuccessResponse creates a new SuccessResponse
func NewSuccessResponse(statusCode int, message string, data interface{}) *SuccessResponse {
	return &SuccessResponse{
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
	}
}

// GenerateSessionID generates a unique session ID.
func GenerateSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// GetTimestamp returns the current timestamp.
func GetTimestamp() time.Time {
	return time.Now()
}