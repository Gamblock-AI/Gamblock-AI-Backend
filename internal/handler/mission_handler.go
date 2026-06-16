package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetTodayMission(c *gin.Context) {
	mission, err := h.services.Mission.GetToday(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "mission_fetch_failed", err.Error())
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
	if err := c.ShouldBindJSON(&input); err != nil || input.MissionNumber < 1 || input.MissionNumber > 5 {
		h.respondError(c, http.StatusBadRequest, "invalid_mission", "Nomor misi harus 1-5")
		return
	}

	mission, err := h.services.Mission.UpdateMission(c.Request.Context(), h.currentUserID(c), input.MissionNumber, input.Completed)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "mission_update_failed", err.Error())
		return
	}
	h.respond(c, http.StatusOK, mission)
}
