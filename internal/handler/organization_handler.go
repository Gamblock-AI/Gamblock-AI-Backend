package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type createOrgInput struct {
	Name string `json:"name" binding:"required"`
}

type joinOrgInput struct {
	GroupCode string `json:"group_code" binding:"required"`
}

func (h *Handler) CreateOrganization(c *gin.Context) {
	var input createOrgInput
	if err := c.ShouldBindJSON(&input); err != nil || input.Name == "" {
		h.respondCode(c, http.StatusBadRequest, "name_required")
		return
	}

	org, err := h.services.Organization.Create(c.Request.Context(), input.Name, h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "create_org_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, org)
}

func (h *Handler) GetOrganization(c *gin.Context) {
	org, err := h.services.Organization.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondErrorErr(c, http.StatusNotFound, "org_not_found", err)
		return
	}
	h.respond(c, http.StatusOK, org)
}

func (h *Handler) JoinOrganization(c *gin.Context) {
	var input joinOrgInput
	if err := c.ShouldBindJSON(&input); err != nil || input.GroupCode == "" {
		h.respondCode(c, http.StatusBadRequest, "group_code_required")
		return
	}

	org, err := h.services.Organization.JoinByCode(c.Request.Context(), input.GroupCode, h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "join_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"joined": true, "organization": org})
}

func (h *Handler) GetCurrentUserOrganization(c *gin.Context) {
	org, err := h.services.Organization.GetByUserID(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondCode(c, http.StatusNotFound, "no_org")
		return
	}
	h.respond(c, http.StatusOK, org)
}

func (h *Handler) ListOrganizationMembers(c *gin.Context) {
	members, err := h.services.Organization.ListMembers(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "list_members_failed", err)
		return
	}
	h.respond(c, http.StatusOK, members)
}

func (h *Handler) GetOrganizationAnalytics(c *gin.Context) {
	analytics, err := h.services.Organization.GetAnalytics(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "analytics_failed", err)
		return
	}
	h.respond(c, http.StatusOK, analytics)
}

func (h *Handler) RemoveOrganizationMember(c *gin.Context) {
	err := h.services.Organization.RemoveMember(
		c.Request.Context(),
		c.Param("id"),
		c.Param("user_id"),
		h.currentUserID(c),
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "remove_member_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"removed": true})
}
