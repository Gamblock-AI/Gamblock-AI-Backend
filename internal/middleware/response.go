package middleware

import (
	"github.com/gamblock-ai/gamblock-ai-backend/internal/i18n"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type envelope struct {
	Data      any       `json:"data"`
	Error     *apiError `json:"error"`
	RequestID string    `json:"request_id"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (m *Middleware) respondError(c *gin.Context, status int, code string) {
	requestID := uuid.NewString()
	if value, ok := c.Get("request_id"); ok {
		if id, ok := value.(string); ok {
			requestID = id
		}
	}
	c.JSON(status, envelope{Data: nil, Error: &apiError{Code: code, Message: i18n.Friendly(code)}, RequestID: requestID})
}
