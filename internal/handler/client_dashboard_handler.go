package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ClientDashboardSummary(c *gin.Context) {
	summary, _, _, err := h.services.Client.Dashboard(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "dashboard_summary_failed", err)
		return
	}
	h.respond(c, http.StatusOK, summary)
}

func (h *Handler) ClientProtectionStatus(c *gin.Context) {
	_, protection, _, err := h.services.Client.Dashboard(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "protection_status_failed", err)
		return
	}
	h.respond(c, http.StatusOK, protection)
}

func (h *Handler) ClientProgressSnapshot(c *gin.Context) {
	_, _, progress, err := h.services.Client.Dashboard(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "progress_snapshot_failed", err)
		return
	}
	h.respond(c, http.StatusOK, progress)
}
