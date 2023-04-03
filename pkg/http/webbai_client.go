package http

import (
	"encoding/json"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/webb-ai/k8s-agent/pkg/api"
	"go.uber.org/atomic"
)

type WebbaiHttpClient struct {
	clientId     string
	clientSecret string
	authUrl      string
	changeUrl    string
	resourceUrl  string
	token        atomic.String
}

func (c *WebbaiHttpClient) SendK8sChangeEvent(event *api.ResourceChangeEvent) error {
	return c.sendRequest(c.changeUrl, event)
}
func (c *WebbaiHttpClient) SendK8sResources(list *api.ResourceList) error {
	return c.sendRequest(c.resourceUrl, list)
}

func (c *WebbaiHttpClient) sendRequest(url string, data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	client := retryablehttp.NewClient()
	response, err := SendRequestWithToken(client, url, c.token.Load(), bytes)
	if err != nil {
		return err
	}
	if response.StatusCode == 401 { // Unauthorized, obtain new token and try again
		err = c.obtainNewToken()
		if err != nil {
			return err
		}
		_, err = SendRequestWithToken(client, url, c.token.Load(), bytes)
		return err
	} else {
		return nil
	}
}

func (c *WebbaiHttpClient) obtainNewToken() error {
	client := retryablehttp.NewClient()
	token, err := GetAccessToken(client, c.authUrl, c.clientId, c.clientSecret, "")
	if err != nil {
		return err
	}
	c.token.Store(token)
	return nil
}
