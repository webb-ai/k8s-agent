package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	//nolint:staticcheck // Ignore error here
	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"

	"k8s.io/klog/v2"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/webb-ai/k8s-agent/pkg/api"
	"go.uber.org/atomic"
)

type AgentInfo struct {
	AgentVersion               string `json:"agent_version"`
	KafkaServer                string `json:"kafka_server"`
	LastChangeCollectionTime   int64  `json:"last_change_collection_time"`
	LastKafkaCollectionTime    int64  `json:"last_kafka_collection_time"`
	LastResourceCollectionTime int64  `json:"last_resource_collection_time"`
	LastTrafficCollectionTime  int64  `json:"last_traffic_collection_time"`
}

type WebbaiHttpClient struct {
	ClientId     string
	ClientSecret string
	AuthUrl      string
	ChangeUrl    string
	ResourceUrl  string
	MetricsUrl   string
	AgentInfoUrl string
	token        atomic.String
	agentInfo    *AgentInfo
}

func NewWebbaiClient(agentVersion, kafkaServer string) api.Client {
	clientId := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("API_KEY")

	if clientId == "" || clientSecret == "" {
		return nil
	}

	agentInfo := &AgentInfo{
		AgentVersion: agentVersion,
		KafkaServer:  kafkaServer,
	}

	client := &WebbaiHttpClient{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		AuthUrl:      "https://api.webb.ai/oauth/token",
		ChangeUrl:    "https://api.webb.ai/k8s_changes",
		ResourceUrl:  "https://api.webb.ai/k8s_resources",
		MetricsUrl:   "https://api.webb.ai/metrics",
		AgentInfoUrl: "https://api.webb.ai/agent_info",
		agentInfo:    agentInfo,
	}
	err := client.obtainNewToken()
	if err != nil {
		klog.Error(err)
		return nil
	}

	return client
}

func (c *WebbaiHttpClient) SendChangeEvent(event *api.ChangeEvent) error {
	klog.Infof("sending change event to %s", c.ChangeUrl)
	if event.EventType == api.KafkaUpdate {
		c.agentInfo.LastKafkaCollectionTime = event.Time
	} else {
		c.agentInfo.LastChangeCollectionTime = event.Time
	}
	err := c.sendRequest(c.ChangeUrl, event)
	return err
}

func (c *WebbaiHttpClient) SendK8sResources(list *api.ResourceList) error {
	klog.Infof("sending k8s resource list to %s", c.ResourceUrl)
	c.agentInfo.LastResourceCollectionTime = list.Time
	err := c.sendRequest(c.ResourceUrl, list)
	return err
}

func (c *WebbaiHttpClient) SendTrafficMetrics(request *prompb.WriteRequest) error {
	c.agentInfo.LastTrafficCollectionTime = time.Now().Unix()
	data, err := proto.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal write request: %V", err)
	}

	// Compress the serialized data using snappy
	compressed := snappy.Encode(nil, data)
	body := bytes.NewReader(compressed)

	req, err := retryablehttp.NewRequest("POST", c.MetricsUrl, body)
	if err != nil {
		return fmt.Errorf("failed to create post request to %s", c.MetricsUrl)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("Authorization", "Bearer "+c.token.Load())

	client := makeHttpClient()

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post to %s", c.MetricsUrl)
	}
	if resp != nil {
		//nolint:staticcheck // SA5001 Ignore error here
		defer resp.Body.Close()
	}

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {
		return fmt.Errorf("failed to read response body: %s", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to post to %s. Error code: %d. Body: %s", c.MetricsUrl, resp.StatusCode, string(respBody))
	}

	klog.Infof("Successfully send traffic metrics")
	return nil
}

func (c *WebbaiHttpClient) SendAgentInfo() error {
	klog.Infof("sending agent info to %s", c.AgentInfoUrl)
	err := c.sendRequest(c.AgentInfoUrl, c.agentInfo)
	if err != nil {
		klog.Error(err)
	}
	return err
}

func (c *WebbaiHttpClient) sendRequest(url string, data interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		klog.Error(err)
		return err
	}
	client := makeHttpClient()
	response, err := SendRequestWithToken(client, url, c.token.Load(), bytes)
	if err != nil {
		klog.Error(err)
		return err
	}
	//nolint:staticcheck // SA5001 Ignore error here
	defer response.Body.Close()
	if response.StatusCode == 401 { // Unauthorized, obtain new token and try again
		err = c.obtainNewToken()
		if err != nil {
			klog.Error(err)
			return err
		}
		response, err = SendRequestWithToken(client, url, c.token.Load(), bytes)
		if response != nil {
			//nolint:staticcheck // SA5001 Ignore error here
			defer response.Body.Close()
		}
		return err
	} else {
		return nil
	}
}

func (c *WebbaiHttpClient) obtainNewToken() error {
	klog.Infof("request a new token from %s", c.AuthUrl)
	client := makeHttpClient()
	token, err := GetAccessToken(client, c.AuthUrl, c.ClientId, c.ClientSecret, c.ClientId)
	if err != nil {
		klog.Error(err)
		return err
	}
	klog.Infof("got access token")
	c.token.Store(token)
	return nil
}

func makeHttpClient() *retryablehttp.Client {
	client := retryablehttp.NewClient()
	client.RetryMax = 10
	return client
}
