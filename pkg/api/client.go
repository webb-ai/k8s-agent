package api

type Client interface {
	SendK8sChangeEvent(*ResourceChangeEvent) error
	SendK8sResources(*ResourceList) error
}
