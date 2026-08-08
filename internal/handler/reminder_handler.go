package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
)

func (h *Handler) GetReminderPreference(c *gin.Context) {
	preference, err := h.services.Reminder.GetPreference(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "reminder_preference_load_failed", err)
		return
	}
	h.respond(c, http.StatusOK, preference)
}

func (h *Handler) UpdateReminderPreference(c *gin.Context) {
	var input struct {
		Enabled   bool   `json:"enabled"`
		LocalTime string `json:"local_time"`
		Timezone  string `json:"timezone"`
		Locale    string `json:"locale"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	preference, err := h.services.Reminder.UpdatePreference(
		c.Request.Context(), h.currentUserID(c), input.Enabled, input.LocalTime, input.Timezone, input.Locale,
	)
	if err != nil {
		code := "reminder_preference_update_failed"
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrReminderPreferenceInvalid) {
			code = "reminder_preference_invalid"
		} else {
			status = http.StatusInternalServerError
		}
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, preference)
}

func (h *Handler) UpsertPushSubscription(c *gin.Context) {
	var input struct {
		Endpoint  string  `json:"endpoint"`
		P256dh    string  `json:"p256dh"`
		AuthKey   string  `json:"auth_key"`
		UserAgent *string `json:"user_agent"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Endpoint == "" || input.P256dh == "" || input.AuthKey == "" {
		h.respondCode(c, http.StatusBadRequest, "push_subscription_invalid")
		return
	}
	subscription, err := h.services.Push.UpsertSubscription(
		c.Request.Context(), h.currentUserID(c), input.Endpoint, input.P256dh, input.AuthKey, input.UserAgent,
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "push_subscription_update_failed", err)
		return
	}
	h.respond(c, http.StatusOK, subscription)
}

func (h *Handler) DeletePushSubscription(c *gin.Context) {
	var input struct {
		Endpoint string `json:"endpoint"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Endpoint == "" {
		h.respondCode(c, http.StatusBadRequest, "push_subscription_invalid")
		return
	}
	if err := h.services.Push.DeleteSubscription(c.Request.Context(), h.currentUserID(c), input.Endpoint); err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "push_subscription_update_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"removed": true})
}
