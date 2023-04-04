package http

import (
	"encoding/json"

	"k8s.io/klog/v2"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/webb-ai/k8s-agent/pkg/api"
	"go.uber.org/atomic"
)

type WebbaiHttpClient struct {
	ClientId     string
	ClientSecret string
	AuthUrl      string
	ChangeUrl    string
	ResourceUrl  string
	token        atomic.String
}

func (c *WebbaiHttpClient) SendK8sChangeEvent(event *api.ResourceChangeEvent) error {
	if c.ChangeUrl != "" {
		klog.Infof("sending k8s change event to %s", c.ChangeUrl)
		return c.sendRequest(c.ChangeUrl, event)
	}
	return nil
}
func (c *WebbaiHttpClient) SendK8sResources(list *api.ResourceList) error {
	if c.ResourceUrl != "" {
		klog.Infof("sending k8s resource list to %s", c.ResourceUrl)
		return c.sendRequest(c.ResourceUrl, list)
	}
	return nil
}

func (c *WebbaiHttpClient) sendRequest(url string, data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		klog.Error(err)
		return err
	}
	client := retryablehttp.NewClient()
	response, err := SendRequestWithToken(client, url, c.token.Load(), bytes)
	if err != nil {
		klog.Error(err)
		return err
	}
	if response.StatusCode == 401 { // Unauthorized, obtain new token and try again
		err = c.obtainNewToken()
		if err != nil {
			klog.Error(err)
			return err
		}
		_, err = SendRequestWithToken(client, url, c.token.Load(), bytes)
		return err
	} else {
		return nil
	}
}

func (c *WebbaiHttpClient) obtainNewToken() error {
	klog.Infof("request a new token from %s", c.AuthUrl)
	client := retryablehttp.NewClient()
	token, err := GetAccessToken(client, c.AuthUrl, c.ClientId, c.ClientSecret, c.ClientId)
	klog.Infof("got access token")
	if err != nil {
		klog.Error(err)
		return err
	}
	c.token.Store(token)
	return nil
}
