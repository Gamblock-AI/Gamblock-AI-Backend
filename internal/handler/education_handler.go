package handler

import (
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

func educationStatus(err error) (int, string) {
	if errors.Is(err, repository.ErrEducationNotFound) {
		return http.StatusNotFound, "module_not_found"
	}
	if errors.Is(err, repository.ErrEducationConflict) {
		return http.StatusConflict, "education_conflict"
	}
	return http.StatusBadRequest, "education_validation_failed"
}

func (h *Handler) AdminModuleDetail(c *gin.Context) {
	module, err := h.services.Education.AdminModule(c.Request.Context(), c.Param("id"))
	if err != nil {
		status, code := educationStatus(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, module)
}

func (h *Handler) UpdateAdminModule(c *gin.Context) {
	var input struct {
		Slug             string                  `json:"slug"`
		ExpectedRevision int                     `json:"expected_revision"`
		Document         model.EducationDocument `json:"document"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	module, err := h.services.Education.UpdateDraft(c.Request.Context(), h.currentUserID(c), c.Param("id"), input.Slug, input.ExpectedRevision, input.Document)
	if err != nil {
		status, code := educationStatus(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, module)
}

func (h *Handler) SubmitAdminModuleReview(c *gin.Context) {
	module, err := h.services.Education.SubmitReview(c.Request.Context(), h.currentUserID(c), c.Param("id"))
	if err != nil {
		status, code := educationStatus(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, module)
}

func (h *Handler) PublishAdminModule(c *gin.Context) {
	module, err := h.services.Education.Publish(c.Request.Context(), h.currentUserID(c), c.Param("id"))
	if err != nil {
		status, code := educationStatus(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, module)
}

func (h *Handler) ArchiveAdminModule(c *gin.Context) {
	module, err := h.services.Education.Archive(c.Request.Context(), h.currentUserID(c), c.Param("id"))
	if err != nil {
		status, code := educationStatus(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, module)
}

func (h *Handler) AdminModuleRevisions(c *gin.Context) {
	items, err := h.services.Education.Revisions(c.Request.Context(), c.Param("id"))
	if err != nil {
		status, code := educationStatus(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, items)
}

func (h *Handler) RollbackAdminModule(c *gin.Context) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	module, err := h.services.Education.Rollback(c.Request.Context(), h.currentUserID(c), c.Param("id"), c.Param("revision_id"), input.Reason)
	if err != nil {
		status, code := educationStatus(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, module)
}

func (h *Handler) UploadAdminEducationMedia(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 101<<20)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.respondCode(c, http.StatusBadRequest, "education_media_invalid")
		return
	}
	defer file.Close() //nolint:errcheck
	media, err := h.services.Education.UploadMedia(c.Request.Context(), h.currentUserID(c), c.PostForm("purpose"), header.Filename, file)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "education_media_invalid", err)
		return
	}
	h.respond(c, http.StatusCreated, media)
}

func (h *Handler) RegisterAdminExternalMedia(c *gin.Context) {
	var input struct {
		Purpose   string `json:"purpose"`
		MediaType string `json:"media_type"`
		URL       string `json:"url"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	media, err := h.services.Education.RegisterExternalMedia(c.Request.Context(), h.currentUserID(c), input.Purpose, input.MediaType, input.URL)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "education_media_invalid", err)
		return
	}
	h.respond(c, http.StatusCreated, media)
}

func (h *Handler) UpdateEducationProgress(c *gin.Context) {
	revision, err := strconv.Atoi(c.Param("revision"))
	if err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	var input model.EducationProgressInput
	if err = c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	progress, err := h.services.Education.UpdateProgress(c.Request.Context(), h.currentUserID(c), c.Param("id"), revision, input)
	if err != nil {
		status, code := educationStatus(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, progress)
}

func (h *Handler) AnswerEducationCheck(c *gin.Context) {
	revision, err := strconv.Atoi(c.Param("revision"))
	if err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	var input model.EducationCheckAnswer
	if err = c.ShouldBindJSON(&input); err != nil || input.ChoiceID == "" {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	result, err := h.services.Education.AnswerCheck(c.Request.Context(), h.currentUserID(c), c.Param("id"), revision, c.Param("check_id"), input.ChoiceID, c.Query("locale"))
	if err != nil {
		status, code := educationStatus(err)
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, result)
}

func (h *Handler) PublishedEducationMedia(c *gin.Context) {
	media, path, err := h.services.Education.MediaFile(c.Request.Context(), c.Param("id"), false)
	if err != nil {
		h.respondCode(c, http.StatusNotFound, "education_media_not_found")
		return
	}
	file, err := os.Open(path)
	if err != nil {
		h.respondCode(c, http.StatusNotFound, "education_media_not_found")
		return
	}
	defer file.Close() //nolint:errcheck
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "public, max-age=86400, immutable")
	c.Header("Content-Disposition", "inline")
	c.DataFromReader(http.StatusOK, media.SizeBytes, media.MIMEType, file, nil)
}

func (h *Handler) AdminEducationMedia(c *gin.Context) {
	media, path, err := h.services.Education.MediaFile(c.Request.Context(), c.Param("id"), true)
	if err != nil {
		h.respondCode(c, http.StatusNotFound, "education_media_not_found")
		return
	}
	file, err := os.Open(path)
	if err != nil {
		h.respondCode(c, http.StatusNotFound, "education_media_not_found")
		return
	}
	defer file.Close() //nolint:errcheck
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, no-store")
	c.DataFromReader(http.StatusOK, media.SizeBytes, media.MIMEType, file, nil)
}
