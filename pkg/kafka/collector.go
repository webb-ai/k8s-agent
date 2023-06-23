package kafka

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/webb-ai/k8s-agent/pkg/api"

	"github.com/Shopify/sarama"
	"k8s.io/klog/v2"
)

// type kafkaSaslType string
//
// type kafkaMetadata struct {
//	bootstrapServers []string
//
//	// SASL
//	saslType kafkaSaslType
//	username string
//	password string
//
//	// OAUTHBEARER
//	scopes                []string
//	oauthTokenEndpointURI string
//	oauthExtensions       map[string]string
//
//	// TLS
//	enableTLS   bool
//	cert        string
//	key         string
//	keyPassword string
//	ca          string
//}

type Collector struct {
	// metadata *kafkaMetadata
	client   api.Client
	admin    sarama.ClusterAdmin
	interval time.Duration
	data     map[string]interface{}
}

func NewKafkaCollector(
	bootstrapServers []string,
	pollingInterval time.Duration,
	client api.Client,
) (*Collector, error) {
	config := sarama.NewConfig()
	admin, err := sarama.NewClusterAdmin(bootstrapServers, config)
	if err != nil {
		return nil, fmt.Errorf("error creating kafka admin: %w", err)
	}
	return &Collector{
		client:   client,
		admin:    admin,
		interval: pollingInterval,
		data:     make(map[string]interface{}),
	}, nil
}

func (c *Collector) Start(ctx context.Context) error {
	klog.Infof("starting to collect kafka changes every %v", c.interval)

	for {
		select {
		case <-time.After(c.interval):
			c.collectKafkaChanges()
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Collector) collectKafkaChanges() {
	klog.Infof("collecting kafka changes")
	newValue, err := c.admin.ListTopics()
	if err != nil {
		klog.Error(err)
		return
	}

	oldValue, ok := c.data["ListTopics"]
	if ok {
		if !reflect.DeepEqual(oldValue, newValue) {
			klog.Infof("detected kafka changes")
			_ = c.client.SendChangeEvent(api.NewKafkaChangeEvent(oldValue, newValue, "ListTopics"))
		}
	}
	c.data["ListTopics"] = newValue
}
