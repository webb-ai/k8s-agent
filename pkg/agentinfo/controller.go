package agentinfo

import (
	"context"
	"time"

	"github.com/webb-ai/k8s-agent/pkg/api"
	"k8s.io/klog/v2"
)

type Controller struct {
	// metadata *kafkaMetadata
	client   api.Client
	interval time.Duration
}

func NewController(
	pollingInterval time.Duration,
	client api.Client,
) *Controller {
	return &Controller{
		client:   client,
		interval: pollingInterval,
	}
}

func (c *Controller) Start(ctx context.Context) error {
	klog.Infof("starting to send agent info every %v", c.interval)
	_ = c.client.SendAgentInfo()
	for {
		select {
		case <-time.After(c.interval):
			_ = c.client.SendAgentInfo()
		case <-ctx.Done():
			return nil
		}
	}
}
