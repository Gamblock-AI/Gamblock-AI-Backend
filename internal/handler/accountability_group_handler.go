package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (h *Handler) AccountabilityWorkspace(c *gin.Context) {
	workspace, err := h.services.AccountabilityGroups.Workspace(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusForbidden, "accountability_workspace_failed", err)
		return
	}
	h.respond(c, http.StatusOK, workspace)
}

func (h *Handler) CreateAccountabilityGroup(c *gin.Context) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	group, err := h.services.AccountabilityGroups.CreateGroup(c.Request.Context(), h.currentUserID(c), input.Name, input.Description)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "accountability_group_create_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, group)
}

func (h *Handler) PreviewAccountabilityGroup(c *gin.Context) {
	var input struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	group, err := h.services.AccountabilityGroups.PreviewJoin(c.Request.Context(), h.currentUserID(c), input.Code)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "accountability_code_invalid", err)
		return
	}
	h.respond(c, http.StatusOK, group)
}

func (h *Handler) JoinAccountabilityGroup(c *gin.Context) {
	var input struct {
		Code      string `json:"code"`
		Confirmed bool   `json:"confirmed"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	membership, err := h.services.AccountabilityGroups.Join(c.Request.Context(), h.currentUserID(c), input.Code, input.Confirmed)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "accountability_join_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, membership)
}

func (h *Handler) RotateAccountabilityGroupCode(c *gin.Context) {
	code, err := h.services.AccountabilityGroups.RotateCode(c.Request.Context(), c.Param("group_id"), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusForbidden, "accountability_code_rotate_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"join_code": code})
}

func (h *Handler) ArchiveAccountabilityGroup(c *gin.Context) {
	if err := h.services.AccountabilityGroups.ArchiveGroup(c.Request.Context(), h.currentUserID(c), c.Param("group_id")); err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "accountability_group_archive_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"archived": true})
}

func (h *Handler) UpdateAccountabilitySharing(c *gin.Context) {
	var input model.SharingPreferences
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	membership, err := h.services.AccountabilityGroups.UpdateSharing(c.Request.Context(), h.currentUserID(c), c.Param("membership_id"), input)
	if err != nil {
		h.respondErrorErr(c, http.StatusForbidden, "accountability_sharing_update_failed", err)
		return
	}
	h.respond(c, http.StatusOK, membership)
}

func (h *Handler) RequestAccountabilityLeave(c *gin.Context) {
	var input struct {
		Kind   string `json:"kind"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	request, err := h.services.AccountabilityGroups.RequestLeave(c.Request.Context(), h.currentUserID(c), c.Param("membership_id"), input.Kind, input.Reason)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "accountability_leave_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, request)
}

func (h *Handler) ResolveAccountabilityLeave(c *gin.Context) {
	var input struct {
		Decision string `json:"decision"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	if err := h.services.AccountabilityGroups.ResolveLeave(c.Request.Context(), h.currentUserID(c), c.Param("request_id"), input.Decision); err != nil {
		h.respondErrorErr(c, http.StatusForbidden, "accountability_leave_resolve_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"resolved": true})
}

func (h *Handler) RemoveAccountabilityMember(c *gin.Context) {
	var input struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&input)
	if err := h.services.AccountabilityGroups.RemoveMember(c.Request.Context(), h.currentUserID(c), c.Param("membership_id"), input.Reason); err != nil {
		h.respondErrorErr(c, http.StatusForbidden, "accountability_member_remove_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"removed": true})
}

func (h *Handler) CreatePartnerContactRequest(c *gin.Context) {
	var input struct {
		MembershipID string `json:"membership_id"`
		Category     string `json:"category"`
		Message      string `json:"message"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	request, err := h.services.AccountabilityGroups.CreateContactRequest(c.Request.Context(), h.currentUserID(c), input.MembershipID, input.Category, input.Message)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "partner_contact_create_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, request)
}

func (h *Handler) TransitionPartnerContactRequest(c *gin.Context) {
	var input struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	if err := h.services.AccountabilityGroups.TransitionContactRequest(c.Request.Context(), h.currentUserID(c), c.Param("request_id"), input.Status); err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "partner_contact_transition_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"updated": true})
}
