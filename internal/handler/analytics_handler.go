package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// AccountabilityAnalytics serves partner-scoped aggregate analytics. Only
// members who consented to sharing protection activity contribute counts.
func (h *Handler) AccountabilityAnalytics(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "14"))
	if err != nil {
		h.respondCode(c, http.StatusBadRequest, "analytics_period_invalid")
		return
	}
	analytics, err := h.services.Accountability.GroupAnalytics(
		c.Request.Context(),
		h.currentUserID(c),
		c.Query("group_id"),
		days,
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "analytics_failed", err)
		return
	}
	h.respond(c, http.StatusOK, analytics)
}

// AdminAnalytics serves platform-wide aggregate analytics to verified admins.
func (h *Handler) AdminAnalytics(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "14"))
	if err != nil {
		h.respondCode(c, http.StatusBadRequest, "analytics_period_invalid")
		return
	}
	analytics, err := h.services.Admin.PlatformAnalytics(c.Request.Context(), days)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "analytics_failed", err)
		return
	}
	h.respond(c, http.StatusOK, analytics)
}
