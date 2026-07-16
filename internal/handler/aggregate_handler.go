package handler

import (
	"net/http"

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
