package api

import "github.com/prometheus/prometheus/prompb"

type Client interface {
	SendK8sChangeEvent(*ResourceChangeEvent) error
	SendK8sResources(*ResourceList) error
	SendTrafficMetrics(*prompb.WriteRequest) error
}

type NoOpClient struct {
}

func (nc *NoOpClient) SendK8sChangeEvent(*ResourceChangeEvent) error {
	return nil
}

func (nc *NoOpClient) SendK8sResources(*ResourceList) error {
	return nil
}

func (nc *NoOpClient) SendTrafficMetrics(*prompb.WriteRequest) error {
	return nil
}
