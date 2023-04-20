package api

type Client interface {
	SendK8sChangeEvent(*ResourceChangeEvent) error
	SendK8sResources(*ResourceList) error
}

type NoOpClient struct {
}

func (nc *NoOpClient) SendK8sChangeEvent(*ResourceChangeEvent) error {
	return nil
}

func (nc *NoOpClient) SendK8sResources(*ResourceList) error {
	return nil
}
