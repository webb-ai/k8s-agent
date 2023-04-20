package http

import (
	"encoding/json"
	"os"

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

func NewWebbaiClient() api.Client {
	clientId := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("API_KEY")

	if clientId == "" || clientSecret == "" {
		return nil
	}

	client := &WebbaiHttpClient{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		AuthUrl:      "https://api.webb.ai/oauth/token",
		ChangeUrl:    "https://api.webb.ai/k8s_changes",
		ResourceUrl:  "https://api.webb.ai/k8s_resources",
	}
	err := client.obtainNewToken()
	if err != nil {
		klog.Error(err)
		return client
	}

	return nil
}

func (c *WebbaiHttpClient) SendK8sChangeEvent(event *api.ResourceChangeEvent) error {
	klog.Infof("sending k8s change event to %s", c.ChangeUrl)
	return c.sendRequest(c.ChangeUrl, event)
}
func (c *WebbaiHttpClient) SendK8sResources(list *api.ResourceList) error {
	klog.Infof("sending k8s resource list to %s", c.ResourceUrl)
	return c.sendRequest(c.ResourceUrl, list)
}

func (c *WebbaiHttpClient) sendRequest(url string, data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		klog.Error(err)
		return err
	}
	client := retryablehttp.NewClient()
	response, err := SendRequestWithToken(client, url, c.token.Load(), bytes)
	//nolint:staticcheck // SA5001 Ignore error here
	defer response.Body.Close()
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
		response, err = SendRequestWithToken(client, url, c.token.Load(), bytes)
		//nolint:staticcheck // SA5001 Ignore error here
		defer response.Body.Close()
		return err
	} else {
		return nil
	}
}

func (c *WebbaiHttpClient) obtainNewToken() error {
	klog.Infof("request a new token from %s", c.AuthUrl)
	client := retryablehttp.NewClient()
	token, err := GetAccessToken(client, c.AuthUrl, c.ClientId, c.ClientSecret, c.ClientId)
	if err != nil {
		klog.Error(err)
		return err
	}
	klog.Infof("got access token")
	c.token.Store(token)
	return nil
}
