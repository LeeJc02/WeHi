package controllers

import "github.com/gin-gonic/gin"

type DocsController struct{}

func NewDocsController() *DocsController {
	return &DocsController{}
}

func (ctl *DocsController) OpenAPI(c *gin.Context) {
	c.File("internal/docs/openapi.yaml")
}
