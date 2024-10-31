package handlers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type RetrieveRequest struct {
	ImageName string `json:"imagename"`
}

type Handler struct {
	ContentService RetrieveContentService
}

func NewHandler(rc RetrieveContentService) *Handler {
	return &Handler{
		ContentService: rc,
	}
}

func (h *Handler) HandleRetrieveContent(c *gin.Context) {
	var req RetrieveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	binaries, err := h.ContentService.RetrieveContent(req.ImageName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, binaries)
}

func (h *Handler) HandleGetBinary(c *gin.Context) {
	name := c.Param("name")
	tag := c.Param("tag")

	filepath, err := h.ContentService.GetBinaries(name, tag)
	if err != nil || filepath == "" {
		c.String(http.StatusNotFound, "File not found")
		return
	}

	file, err := os.Open(filepath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error getting file: %v", err)})
		return
	}

	fileInfo, err := file.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error getting file: %v", err)})
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", name))
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	c.File(filepath)
}
