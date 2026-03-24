package controllers

import (
	"io"
	"net/http"
	"os"
	"time"

	httpx "github.com/LeeJc02/WeHi/backend/internal/platform/httpx"
	"github.com/LeeJc02/WeHi/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
)

type UploadService interface {
	Presign(req contracts.UploadPresignRequest) (*contracts.UploadPresignResponse, error)
	PutObject(objectKey string, reader io.Reader) error
	Complete(req contracts.UploadCompleteRequest) (*contracts.UploadCompleteResponse, error)
	Open(objectKey string) (*os.File, string, error)
}

type UploadController struct {
	service UploadService
}

func NewUploadController(service UploadService) *UploadController {
	return &UploadController{service: service}
}

func (ctl *UploadController) Presign(c *gin.Context) {
	_, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	var req contracts.UploadPresignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	resp, err := ctl.service.Presign(req)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, resp)
}

func (ctl *UploadController) PutObject(c *gin.Context) {
	_, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	if err := ctl.service.PutObject(c.Param("key"), c.Request.Body); err != nil {
		httpx.FailError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctl *UploadController) Complete(c *gin.Context) {
	_, _, ok := requireCurrentUser(c)
	if !ok {
		return
	}
	var req contracts.UploadCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, 400, err.Error())
		return
	}
	resp, err := ctl.service.Complete(req)
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	httpx.Success(c, resp)
}

func (ctl *UploadController) Download(c *gin.Context) {
	file, contentType, err := ctl.service.Open(c.Param("key"))
	if err != nil {
		httpx.FailError(c, err)
		return
	}
	defer file.Close()
	if contentType != "" {
		c.Header("Content-Type", contentType)
	}
	http.ServeContent(c.Writer, c.Request, c.Param("key"), time.Time{}, file)
}
