package api

import "github.com/prometheus/prometheus/prompb"

type Client interface {
	SendChangeEvent(*ChangeEvent) error
	SendK8sResources(*ResourceList) error
	SendTrafficMetrics(*prompb.WriteRequest) error
	SendAgentInfo() error
}

type NoOpClient struct {
}

func (nc *NoOpClient) SendChangeEvent(*ChangeEvent) error {
	return nil
}

func (nc *NoOpClient) SendK8sResources(*ResourceList) error {
	return nil
}

func (nc *NoOpClient) SendTrafficMetrics(*prompb.WriteRequest) error {
	return nil
}

func (nc *NoOpClient) SendAgentInfo() error {
	return nil
}
