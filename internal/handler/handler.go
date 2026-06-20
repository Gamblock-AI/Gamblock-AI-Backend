package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/i18n"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/middleware"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
)

type Handler struct {
	cfg        config.Config
	services   *service.Container
	middleware *middleware.Middleware
	logger     *zap.Logger
}

type envelope struct {
	Data      any       `json:"data"`
	Error     *apiError `json:"error"`
	RequestID string    `json:"request_id"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func New(services *service.Container, middleware *middleware.Middleware, cfg config.Config, logger *zap.Logger) *Handler {
	return &Handler{
		services:   services,
		middleware: middleware,
		cfg:        cfg,
		logger:     logger,
	}
}

func (h *Handler) Health(c *gin.Context) {
	h.respond(c, http.StatusOK, gin.H{"status": "ok", "service": "gamblock-ai-backend"})
}

func (h *Handler) Ready(c *gin.Context) {
	ready := gin.H{"status": "ready", "database_configured": h.cfg.DatabaseURL != "", "storage": h.cfg.ArtifactStoragePath}
	h.respond(c, http.StatusOK, ready)
}

func (h *Handler) respond(c *gin.Context, status int, data any) {
	c.JSON(status, envelope{Data: data, Error: nil, RequestID: h.requestID(c)})
}

func (h *Handler) respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, envelope{Data: nil, Error: &apiError{Code: code, Message: message}, RequestID: h.requestID(c)})
}

// respondCode responds with the friendly message from the i18n catalog for the
// given code. Use this for validation/hint errors that have no underlying Go
// error. Production always shows the friendly catalog message; development
// appends the code for easier debugging.
func (h *Handler) respondCode(c *gin.Context, status int, code string) {
	msg := i18n.Friendly(code)
	if !h.cfg.IsProduction() {
		msg = "[" + code + "] " + msg
	}
	c.JSON(status, envelope{Data: nil, Error: &apiError{Code: code, Message: msg}, RequestID: h.requestID(c)})
}

// respondErrorErr responds with an env-gated message derived from [err].
//
// Production: the friendly catalog message for [code] (never [err].Error()),
// so internal details never leak to clients. Development: the technical
// err.Error() for debugging. The full technical error is always logged with the
// request id so it is never lost.
func (h *Handler) respondErrorErr(c *gin.Context, status int, code string, err error) {
	reqID := h.requestID(c)
	if h.logger != nil && err != nil {
		h.logger.Warn("request failed",
			zap.String("request_id", reqID),
			zap.String("code", code),
			zap.Int("status", status),
			zap.Error(err),
		)
	}
	msg := i18n.Friendly(code)
	if !h.cfg.IsProduction() && err != nil {
		msg = "[" + code + "] " + err.Error()
	}
	c.JSON(status, envelope{Data: nil, Error: &apiError{Code: code, Message: msg}, RequestID: reqID})
}

func (h *Handler) requestID(c *gin.Context) string {
	if value, ok := c.Get("request_id"); ok {
		if id, ok := value.(string); ok {
			return id
		}
	}
	return uuid.NewString()
}

func (h *Handler) currentUserID(c *gin.Context) string {
	val, _ := c.Get("user_id")
	if id, ok := val.(string); ok {
		return id
	}
	return ""
}
