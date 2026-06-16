package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetPartners(c *gin.Context) {
	activePartner, items, err := h.services.Accountability.GetPartners(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "fetch_partners_failed", err.Error())
		return
	}
	h.respond(c, http.StatusOK, gin.H{
		"active_partner": activePartner,
		"items":          items,
	})
}

func (h *Handler) CreatePartnerInvitation(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.Email == "" {
		h.respondError(c, http.StatusBadRequest, "partner_email_required", "Partner email is required")
		return
	}

	invite, err := h.services.Accountability.CreatePartnerInvitation(c.Request.Context(), h.currentUserID(c), input.Email, input.Phone)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "partner_invite_failed", err.Error())
		return
	}
	h.respond(c, http.StatusCreated, invite)
}

func (h *Handler) AcceptPartnerInvitation(c *gin.Context) {
	token := c.Param("token")
	err := h.services.Accountability.AcceptInvitation(c.Request.Context(), token, h.currentUserID(c))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "partner_accept_failed", err.Error())
		return
	}
	h.respond(c, http.StatusOK, gin.H{"accepted": true})
}

func (h *Handler) RevokePartner(c *gin.Context) {
	err := h.services.Accountability.RevokePartner(c.Request.Context(), c.Param("partner_link_id"))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "partner_revoke_failed", err.Error())
		return
	}
	h.respond(c, http.StatusOK, gin.H{"revoked": true})
}

func (h *Handler) GetApprovalRequests(c *gin.Context) {
	requests, err := h.services.Accountability.GetApprovalRequests(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "fetch_approval_requests_failed", err.Error())
		return
	}
	h.respond(c, http.StatusOK, requests)
}

func (h *Handler) CreateApprovalRequest(c *gin.Context) {
	var input struct {
		Action                   string `json:"action"`
		Reason                   string `json:"reason"`
		RequestedDurationMinutes int    `json:"requested_duration_minutes"`
		DeviceID                 string `json:"device_id"`
		PartnerLinkID            string `json:"partner_link_id"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.Action == "" {
		h.respondError(c, http.StatusBadRequest, "action_required", "Action is required")
		return
	}

	err := h.services.Accountability.CreateApprovalRequest(
		c.Request.Context(),
		h.currentUserID(c),
		input.DeviceID,
		input.PartnerLinkID,
		input.Action,
		input.Reason,
		input.RequestedDurationMinutes,
	)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "approval_request_failed", err.Error())
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"requested": true})
}

func (h *Handler) CancelApprovalRequest(c *gin.Context) {
	err := h.services.Accountability.ResolveApprovalRequest(c.Request.Context(), c.Param("id"), "cancelled", h.currentUserID(c))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "approval_cancel_failed", err.Error())
		return
	}
	h.respond(c, http.StatusOK, gin.H{"cancelled": true})
}

func (h *Handler) ApproveApprovalRequest(c *gin.Context) {
	err := h.services.Accountability.ResolveApprovalRequest(c.Request.Context(), c.Param("id"), "approved", h.currentUserID(c))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "approval_approve_failed", err.Error())
		return
	}
	h.respond(c, http.StatusOK, gin.H{"approved": true})
}

func (h *Handler) DenyApprovalRequest(c *gin.Context) {
	err := h.services.Accountability.ResolveApprovalRequest(c.Request.Context(), c.Param("id"), "denied", h.currentUserID(c))
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "approval_deny_failed", err.Error())
		return
	}
	h.respond(c, http.StatusOK, gin.H{"denied": true})
}
