package traffic

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

// ScrapeTarget scrapes an http endpoint for prometheus metrics
func ScrapeTarget(targetURL string) (map[string]*dto.MetricFamily, error) {
	resp, err := http.Get(targetURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching metrics from target: %v", err)
	}
	//nolint:staticcheck // SA5001 Ignore error here
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	parser := expfmt.TextParser{}

	return parser.TextToMetricFamilies(bytes.NewReader(body))
}
