package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
)

func (h *Handler) LearningHubCatalog(c *gin.Context) {
	data, err := h.services.LearningHub.Catalog(c.Request.Context(), h.currentUserID(c), c.Query("locale"))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "learning_hub_fetch_failed", err)
		return
	}
	h.respond(c, http.StatusOK, data)
}

func (h *Handler) LearningHubItem(c *gin.Context) {
	item, err := h.services.LearningHub.Item(c.Request.Context(), h.currentUserID(c), c.Param("slug"), c.Query("locale"))
	if err != nil {
		status, code := learningHubError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) LearningHubProgress(c *gin.Context) {
	progress, err := h.services.LearningHub.Progress(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "learning_hub_progress_failed", err)
		return
	}
	h.respond(c, http.StatusOK, progress)
}

func (h *Handler) UpdateLearningHubState(c *gin.Context) {
	var input model.LearningStateInput
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.State) == "" {
		h.respondCode(c, http.StatusBadRequest, "learning_hub_state_invalid")
		return
	}
	progress, err := h.services.LearningHub.SaveState(c.Request.Context(), h.currentUserID(c), c.Param("id"), input.State)
	if err != nil {
		status, code := learningHubError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, progress)
}

func (h *Handler) CreateLearningHubCheckpoint(c *gin.Context) {
	var input model.LearningCheckpointInput
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "learning_hub_checkpoint_invalid")
		return
	}
	result, err := h.services.LearningHub.Checkpoint(c.Request.Context(), h.currentUserID(c), c.Param("id"), input)
	if err != nil {
		status, code := learningHubError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, result)
}

func learningHubError(err error) (int, string) {
	switch {
	case errors.Is(err, service.ErrLearningHubStateInvalid), errors.Is(err, repository.ErrLearningStateInvalid):
		return http.StatusBadRequest, "learning_hub_state_invalid"
	case errors.Is(err, service.ErrLearningHubCheckpointInvalid), errors.Is(err, repository.ErrLearningCheckpointInvalid):
		return http.StatusBadRequest, "learning_hub_checkpoint_invalid"
	case errors.Is(err, repository.ErrLearningItemNotFound):
		return http.StatusNotFound, "learning_hub_item_not_found"
	default:
		return http.StatusInternalServerError, "learning_hub_mutation_failed"
	}
}

func learningHubAdminError(err error) (int, string) {
	switch {
	case errors.Is(err, service.ErrLearningHubAdminInvalid), errors.Is(err, service.ErrLearningHubTransitionInvalid), errors.Is(err, repository.ErrLearningAdminInvalidStatus):
		return http.StatusBadRequest, "learning_hub_admin_validation_failed"
	case errors.Is(err, service.ErrLearningHubAdminConflict), errors.Is(err, repository.ErrLearningAdminConflict):
		return http.StatusConflict, "learning_hub_admin_conflict"
	case errors.Is(err, service.ErrLearningHubTaxonomyConflict), errors.Is(err, repository.ErrLearningTaxonomyConflict):
		return http.StatusConflict, "learning_hub_taxonomy_conflict"
	case errors.Is(err, service.ErrLearningHubAdminNotFound), errors.Is(err, repository.ErrLearningAdminNotFound):
		return http.StatusNotFound, "learning_hub_admin_not_found"
	default:
		return http.StatusInternalServerError, "learning_hub_admin_failed"
	}
}

func (h *Handler) AdminLearningHubItems(c *gin.Context) {
	items, err := h.services.LearningHub.AdminItems(c.Request.Context(), c.Query("status"))
	if err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, items)
}

func (h *Handler) AdminLearningHubItem(c *gin.Context) {
	item, err := h.services.LearningHub.AdminItem(c.Request.Context(), c.Param("id"))
	if err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) CreateAdminLearningHubItem(c *gin.Context) {
	var input model.LearningItemDraft
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "learning_hub_admin_validation_failed")
		return
	}
	item, err := h.services.LearningHub.CreateAdminItem(c.Request.Context(), h.currentUserID(c), input)
	if err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusCreated, item)
}

func (h *Handler) UpdateAdminLearningHubItem(c *gin.Context) {
	var input struct {
		ExpectedRevision int                     `json:"expected_revision"`
		Draft            model.LearningItemDraft `json:"draft"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "learning_hub_admin_validation_failed")
		return
	}
	item, err := h.services.LearningHub.UpdateAdminItem(c.Request.Context(), h.currentUserID(c), c.Param("id"), input.ExpectedRevision, input.Draft)
	if err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) TransitionAdminLearningHubItem(c *gin.Context) {
	h.transitionAdminLearningHubItem(c, c.Param("action"))
}

func (h *Handler) SubmitAdminLearningHubItem(c *gin.Context) {
	h.transitionAdminLearningHubItem(c, "submit-review")
}

func (h *Handler) PublishAdminLearningHubItem(c *gin.Context) {
	h.transitionAdminLearningHubItem(c, "publish")
}

func (h *Handler) ArchiveAdminLearningHubItem(c *gin.Context) {
	h.transitionAdminLearningHubItem(c, "archive")
}

func (h *Handler) transitionAdminLearningHubItem(c *gin.Context, action string) {
	var (
		item model.AdminLearningItem
		err  error
	)
	actor, id := h.currentUserID(c), c.Param("id")
	switch action {
	case "submit-review":
		item, err = h.services.LearningHub.SubmitAdminItemReview(c.Request.Context(), actor, id)
	case "publish":
		item, err = h.services.LearningHub.PublishAdminItem(c.Request.Context(), actor, id)
	case "archive":
		item, err = h.services.LearningHub.ArchiveAdminItem(c.Request.Context(), actor, id)
	default:
		err = service.ErrLearningHubTransitionInvalid
	}
	if err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) AdminLearningHubRevisions(c *gin.Context) {
	revisions, err := h.services.LearningHub.AdminRevisions(c.Request.Context(), c.Param("id"))
	if err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, revisions)
}

func (h *Handler) RollbackAdminLearningHubItem(c *gin.Context) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "learning_hub_admin_validation_failed")
		return
	}
	item, err := h.services.LearningHub.RollbackAdminItem(c.Request.Context(), h.currentUserID(c), c.Param("id"), c.Param("revision_id"), input.Reason)
	if err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) AdminLearningHubTaxonomy(c *gin.Context) {
	taxonomy, err := h.services.LearningHub.Taxonomy(c.Request.Context())
	if err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, taxonomy)
}

func (h *Handler) CreateAdminLearningHubCluster(c *gin.Context) {
	var input model.LearningClusterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "learning_hub_admin_validation_failed")
		return
	}
	cluster, err := h.services.LearningHub.CreateCluster(c.Request.Context(), h.currentUserID(c), input)
	if err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusCreated, cluster)
}

func (h *Handler) UpdateAdminLearningHubCluster(c *gin.Context) {
	var input model.LearningClusterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "learning_hub_admin_validation_failed")
		return
	}
	cluster, err := h.services.LearningHub.UpdateCluster(c.Request.Context(), h.currentUserID(c), c.Param("id"), input)
	if err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, cluster)
}

func (h *Handler) DeleteAdminLearningHubCluster(c *gin.Context) {
	if err := h.services.LearningHub.DeleteCluster(c.Request.Context(), h.currentUserID(c), c.Param("id")); err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"archived": true})
}

func (h *Handler) CreateAdminLearningHubProgram(c *gin.Context) {
	var input model.AcademicProgramInput
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "learning_hub_admin_validation_failed")
		return
	}
	program, err := h.services.LearningHub.CreateProgram(c.Request.Context(), h.currentUserID(c), input)
	if err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusCreated, program)
}

func (h *Handler) UpdateAdminLearningHubProgram(c *gin.Context) {
	var input model.AcademicProgramInput
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "learning_hub_admin_validation_failed")
		return
	}
	program, err := h.services.LearningHub.UpdateProgram(c.Request.Context(), h.currentUserID(c), c.Param("id"), input)
	if err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, program)
}

func (h *Handler) DeleteAdminLearningHubProgram(c *gin.Context) {
	if err := h.services.LearningHub.DeleteProgram(c.Request.Context(), h.currentUserID(c), c.Param("id")); err != nil {
		status, code := learningHubAdminError(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"archived": true})
}
