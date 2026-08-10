package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

// GetSpkRecommendation returns the SPK-driven daily feature recommendation for
// the authenticated student, including data-sufficiency state and optional LLM
// personalization.
func (h *Handler) GetSpkRecommendation(c *gin.Context) {
	recommendation, err := h.services.Spk.Recommend(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "spk_recommendation_failed", err)
		return
	}
	h.respond(c, http.StatusOK, recommendation)
}

// CompleteSpkIntervention marks the daily recommendation as completed.
func (h *Handler) CompleteSpkIntervention(c *gin.Context) {
	record, err := h.services.Spk.MarkCompleted(c.Request.Context(), h.currentUserID(c), c.Param("id"))
	if err != nil {
		h.respondErrorErr(c, http.StatusNotFound, "spk_intervention_not_found", err)
		return
	}
	h.respond(c, http.StatusOK, record)
}

// GetSpkPreference returns the opt-in SPK settings (default-off LLM flag).
func (h *Handler) GetSpkPreference(c *gin.Context) {
	preference, err := h.services.Spk.GetPreference(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "spk_recommendation_failed", err)
		return
	}
	h.respond(c, http.StatusOK, preference)
}

// UpdateSpkPreference stores the SPK privacy set (master switch, per-category
// data usage, and the LLM personalization opt-in).
func (h *Handler) UpdateSpkPreference(c *gin.Context) {
	var input struct {
		SpkRecommendationEnabled  bool `json:"spk_recommendation_enabled"`
		SpkUseProtection          bool `json:"spk_use_protection"`
		SpkUseRecovery            bool `json:"spk_use_recovery"`
		SpkUsePersonal            bool `json:"spk_use_personal"`
		LLMPersonalizationEnabled bool `json:"llm_personalization_enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	preference, err := h.services.Spk.UpdatePreference(c.Request.Context(), h.currentUserID(c), model.SpkPreference{
		SpkRecommendationEnabled:  input.SpkRecommendationEnabled,
		SpkUseProtection:          input.SpkUseProtection,
		SpkUseRecovery:            input.SpkUseRecovery,
		SpkUsePersonal:            input.SpkUsePersonal,
		LLMPersonalizationEnabled: input.LLMPersonalizationEnabled,
	})
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "spk_recommendation_failed", err)
		return
	}
	h.respond(c, http.StatusOK, preference)
}
