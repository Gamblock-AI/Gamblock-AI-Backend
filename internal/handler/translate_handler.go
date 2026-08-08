package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/deepseek"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
)

type adminTranslateInput struct {
	Texts      []string `json:"texts" binding:"required,min=1,max=50,dive,max=2000"`
	SourceLang string   `json:"source_lang" binding:"required,oneof=id en"`
	TargetLang string   `json:"target_lang" binding:"required,oneof=id en"`
}

type adminTranslateResponse struct {
	Translations []string `json:"translations"`
}

func (h *Handler) AdminTranslate(c *gin.Context) {
	var input adminTranslateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "translation_invalid_input")
		return
	}

	translations, err := h.services.DeepSeek.BatchTranslate(
		c.Request.Context(),
		input.Texts,
		input.SourceLang,
		input.TargetLang,
	)
	if err != nil {
		status := http.StatusInternalServerError
		code := "translation_failed"

		switch {
		case errors.Is(err, service.ErrSameLanguage):
			status = http.StatusBadRequest
			code = "translation_invalid_input"
		case errors.Is(err, service.ErrInvalidLanguage):
			status = http.StatusBadRequest
			code = "translation_invalid_input"
		case errors.Is(err, service.ErrTranslationEmptyText):
			status = http.StatusBadRequest
			code = "translation_invalid_input"
		case errors.Is(err, service.ErrTranslationMaxTexts):
			status = http.StatusBadRequest
			code = "translation_invalid_input"
		case errors.Is(err, service.ErrTextTooLong):
			status = http.StatusBadRequest
			code = "translation_invalid_input"
		case errors.Is(err, deepseek.ErrInvalidAPIKey):
			code = "translation_unavailable"
		case errors.Is(err, deepseek.ErrRateLimited):
			status = http.StatusTooManyRequests
			code = "translation_rate_limited"
		case errors.Is(err, deepseek.ErrServiceUnavailable):
			status = http.StatusServiceUnavailable
			code = "translation_unavailable"
		}

		h.respondErrorErr(c, status, code, err)
		return
	}

	h.respond(c, http.StatusOK, adminTranslateResponse{Translations: translations})
}
