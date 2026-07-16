package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RecordAggregateEvent(c *gin.Context) {
	var input struct {
		DeviceID       string `json:"device_id"`
		EventType      string `json:"event_type"`
		EventDate      string `json:"event_date"`
		Count          int    `json:"count"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	event, err := h.services.Client.RecordAggregate(c.Request.Context(), h.currentUserID(c), input.DeviceID, input.EventType, input.EventDate, input.IdempotencyKey, input.Count)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "aggregate_event_rejected", err)
		return
	}
	h.respond(c, http.StatusAccepted, event)
}

func (h *Handler) ProtectionAnalytics(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "7"))
	if err != nil {
		h.respondCode(c, http.StatusBadRequest, "analytics_period_invalid")
		return
	}
	analytics, err := h.services.Client.ProtectionAnalytics(
		c.Request.Context(),
		h.currentUserID(c),
		c.Query("device_id"),
		days,
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "protection_analytics_failed", err)
		return
	}
	h.respond(c, http.StatusOK, analytics)
}
