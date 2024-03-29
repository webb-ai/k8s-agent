package traffic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/go-retryablehttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	corev1 "k8s.io/api/core/v1"
)

// ScrapeTarget scrapes an http endpoint for prometheus metrics
func ScrapeTarget(targetURL string) (string, map[string]*dto.MetricFamily, error) {
	resp, err := retryablehttp.Get(targetURL)
	if err != nil {
		return "", nil, fmt.Errorf("error fetching metrics from target: %w", err)
	}
	//nolint:staticcheck // SA5001 Ignore error here
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("error reading response body: %w", err)
	}

	parser := expfmt.TextParser{}

	metricFamily, err := parser.TextToMetricFamilies(bytes.NewReader(body))
	return string(body), metricFamily, err
}

// SetPodTargets tells traffic collector which pods to target
func SetPodTargets(pods []*corev1.Pod, targetUrl string) error {
	podsMarshalled, err := json.Marshal(pods)
	if err != nil {
		return fmt.Errorf("error marshalling pods to json: %w", err)
	}
	resp, err := retryablehttp.Post(targetUrl, "application/json", bytes.NewBuffer(podsMarshalled))
	if err != nil {
		return fmt.Errorf("error setting pod target at %s: %w", targetUrl, err)
	}

	//nolint:staticcheck // SA5001 Ignore error here
	defer resp.Body.Close()
	return nil
}
