package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RecordAggregateEvent(c *gin.Context) {
	var input struct {
		DeviceID          string         `json:"device_id"`
		EventType         string         `json:"event_type"`
		EventDate         string         `json:"event_date"`
		Count             int            `json:"count"`
		IdempotencyKey    string         `json:"idempotency_key"`
		Snapshot          bool           `json:"snapshot"`
		MetadataJSON      map[string]any `json:"metadata_json"`
		BlockedEventTimes []string       `json:"blocked_event_times"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	var blockedEventTimes []time.Time
	if len(input.BlockedEventTimes) > 0 {
		var parseErr error
		blockedEventTimes, parseErr = parseBlockedEventTimes(input.BlockedEventTimes)
		if parseErr != nil {
			h.respondErrorErr(c, http.StatusBadRequest, "blocked_events_rejected", parseErr)
			return
		}
	}
	event, err := h.services.Client.RecordAggregate(c.Request.Context(), h.currentUserID(c), input.DeviceID, input.EventType, input.EventDate, input.IdempotencyKey, input.Count, input.Snapshot, input.MetadataJSON)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "aggregate_event_rejected", err)
		return
	}
	if len(blockedEventTimes) > 0 {
		if saveErr := h.services.Client.SaveBlockedEvents(c.Request.Context(), h.currentUserID(c), input.DeviceID, blockedEventTimes); saveErr != nil {
			h.respondErrorErr(c, http.StatusBadRequest, "blocked_events_rejected", saveErr)
			return
		}
	}
	h.respond(c, http.StatusAccepted, event)
}

// parseBlockedEventTimes converts RFC3339 strings to UTC timestamps. It is a
// strict, bounded parser; any invalid entry rejects the whole batch.
func parseBlockedEventTimes(raw []string) ([]time.Time, error) {
	out := make([]time.Time, 0, len(raw))
	for _, value := range raw {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, err
		}
		out = append(out, parsed.UTC())
	}
	return out, nil
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
