package server

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/webb-ai/k8s-agent/pkg/api"
	"k8s.io/klog/v2"
)

type ApiServerProxy struct {
	client api.Client
	port   string
}

func (p *ApiServerProxy) Start(ctx context.Context) error {

	app := gin.Default()
	app.GET("/status", func(c *gin.Context) {
		c.String(http.StatusOK, "running")
	})

	app.GET("/alertmanager", func(c *gin.Context) {
		c.String(http.StatusOK, "alertmanager")
	})

	app.POST("/alertmanager", func(c *gin.Context) {
		payload, err := io.ReadAll(c.Request.Body)
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		issue := &api.IssueRequest{
			IssueSource: "alertmanager",
			Data:        string(payload),
		}
		_ = p.client.SendIssue(issue)
		klog.Infof("issue payload: %s", issue)
		c.String(http.StatusOK, "okay")
	})

	if err := app.Run(p.port); err != nil {
		klog.Error(err)
	}
	return nil
}

func NewApiServerProxy(client api.Client, port string) *ApiServerProxy {
	return &ApiServerProxy{
		client: client,
		port:   port,
	}
}
