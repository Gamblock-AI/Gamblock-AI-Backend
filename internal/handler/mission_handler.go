package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetTodayMission(c *gin.Context) {
	mission, err := h.services.Mission.GetToday(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "mission_fetch_failed", err)
		return
	}
	h.respond(c, http.StatusOK, mission)
}

type updateMissionInput struct {
	MissionNumber int  `json:"mission_number" binding:"required"`
	Completed     bool `json:"completed"`
}

func (h *Handler) UpdateMission(c *gin.Context) {
	var input updateMissionInput
	if err := c.ShouldBindJSON(&input); err != nil || input.MissionNumber < 1 || input.MissionNumber > 6 {
		h.respondCode(c, http.StatusBadRequest, "invalid_mission")
		return
	}

	mission, err := h.services.Mission.UpdateMission(c.Request.Context(), h.currentUserID(c), input.MissionNumber, input.Completed)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "mission_update_failed", err)
		return
	}
	h.respond(c, http.StatusOK, mission)
}

type claimMissionInput struct {
	MissionID string `json:"mission_id" binding:"required"`
}

func (h *Handler) ClaimMission(c *gin.Context) {
	var input claimMissionInput
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.MissionID) == "" {
		h.respondCode(c, http.StatusBadRequest, "invalid_mission")
		return
	}

	mission, err := h.services.Mission.ClaimMissionByID(
		c.Request.Context(),
		h.currentUserID(c),
		input.MissionID,
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusConflict, "mission_update_failed", err)
		return
	}
	h.respond(c, http.StatusOK, mission)
}

type createCustomMissionInput struct {
	Title string `json:"title" binding:"required"`
}

func (h *Handler) CreateCustomMission(c *gin.Context) {
	var input createCustomMissionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "custom_mission_invalid")
		return
	}
	mission, err := h.services.Mission.CreateCustomMission(c.Request.Context(), h.currentUserID(c), input.Title)
	if err != nil {
		h.respondErrorErr(c, missionMutationStatus(err), missionMutationCode(err), err)
		return
	}
	h.respond(c, http.StatusCreated, mission)
}

func (h *Handler) UpdateCustomMission(c *gin.Context) {
	var input createCustomMissionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "custom_mission_invalid")
		return
	}
	mission, err := h.services.Mission.UpdateCustomMission(c.Request.Context(), h.currentUserID(c), c.Param("id"), input.Title)
	if err != nil {
		h.respondErrorErr(c, missionMutationStatus(err), missionMutationCode(err), err)
		return
	}
	h.respond(c, http.StatusOK, mission)
}

func (h *Handler) DeleteCustomMission(c *gin.Context) {
	mission, err := h.services.Mission.DeleteCustomMission(c.Request.Context(), h.currentUserID(c), c.Param("id"))
	if err != nil {
		h.respondErrorErr(c, missionMutationStatus(err), missionMutationCode(err), err)
		return
	}
	h.respond(c, http.StatusOK, mission)
}

func missionMutationCode(err error) string {
	switch {
	case errors.Is(err, service.ErrCustomMissionLimit):
		return "custom_mission_limit"
	case errors.Is(err, service.ErrCustomMissionInvalid):
		return "custom_mission_invalid"
	case errors.Is(err, service.ErrCustomMissionNotEditable):
		return "custom_mission_not_editable"
	default:
		return "mission_update_failed"
	}
}

func missionMutationStatus(err error) int {
	if errors.Is(err, service.ErrCustomMissionInvalid) {
		return http.StatusBadRequest
	}
	return http.StatusConflict
}
