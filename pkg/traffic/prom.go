package traffic

import (
	"fmt"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/prompb"
)

func MetricFamiliesToProtoWriteRequest(metricFamilies map[string]*dto.MetricFamily) *prompb.WriteRequest {
	writeRequest := &prompb.WriteRequest{
		Timeseries: make([]prompb.TimeSeries, 0, len(metricFamilies)),
	}

	for _, family := range metricFamilies {
		if family.GetType() == dto.MetricType_COUNTER {
			appendCounterToWriteRequest(family, writeRequest)
		}

		if family.GetType() == dto.MetricType_HISTOGRAM {
			appendHistogramToWriteRequest(family, writeRequest)
		}
	}

	return writeRequest
}

func ExtractLabels(metric *dto.Metric) []prompb.Label {
	labels := make([]prompb.Label, 0, len(metric.Label))

	for _, pair := range metric.GetLabel() {
		if *pair.Name != "" && *pair.Value != "" {
			labels = append(labels, prompb.Label{
				Name:  *pair.Name,
				Value: *pair.Value,
			})
		}
	}
	return labels
}

func appendCounterToWriteRequest(family *dto.MetricFamily, wr *prompb.WriteRequest) {
	for _, metric := range family.GetMetric() {
		labels := ExtractLabels(metric)
		labels = append(labels, prompb.Label{
			Name:  "__name__",
			Value: family.GetName(),
		})

		samples := []prompb.Sample{
			{
				Value:     metric.GetCounter().GetValue(),
				Timestamp: time.Now().UnixMilli(),
			},
		}

		wr.Timeseries = append(wr.Timeseries, prompb.TimeSeries{
			Labels:  labels,
			Samples: samples,
		})
	}
}

func appendHistogramToWriteRequest(family *dto.MetricFamily, wr *prompb.WriteRequest) {
	for _, metric := range family.GetMetric() {
		labels := ExtractLabels(metric)

		// hist count
		countSamples := []prompb.Sample{
			{
				Value:     float64(metric.GetHistogram().GetSampleCount()),
				Timestamp: time.Now().UnixMilli(),
			},
		}
		//nolint:gocritic
		countLabels := append(labels, prompb.Label{
			Name:  "__name__",
			Value: fmt.Sprintf("%s_count", family.GetName()),
		})

		wr.Timeseries = append(wr.Timeseries, prompb.TimeSeries{
			Labels:  countLabels,
			Samples: countSamples,
		})
		// hist sum
		sumSamples := []prompb.Sample{
			{
				Value:     float64(metric.GetHistogram().GetSampleSum()),
				Timestamp: time.Now().UnixMilli(),
			},
		}

		//nolint:gocritic
		sumLabels := append(labels, prompb.Label{
			Name:  "__name__",
			Value: fmt.Sprintf("%s_sum", family.GetName()),
		})

		wr.Timeseries = append(wr.Timeseries, prompb.TimeSeries{
			Labels:  sumLabels,
			Samples: sumSamples,
		})
		// hist bucket
		for _, bucket := range metric.GetHistogram().GetBucket() {
			bucketSamples := []prompb.Sample{
				{
					Value:     float64(bucket.GetCumulativeCount()),
					Timestamp: time.Now().UnixMilli(),
				},
			}
			//nolint:gocritic
			bucketLabels := append(labels, prompb.Label{
				Name:  "le",
				Value: fmt.Sprintf("%g", bucket.GetUpperBound()),
			})
			bucketLabels = append(bucketLabels, prompb.Label{
				Name:  "__name__",
				Value: fmt.Sprintf("%s_bucket", family.GetName()),
			})

			wr.Timeseries = append(wr.Timeseries, prompb.TimeSeries{
				Labels:  bucketLabels,
				Samples: bucketSamples,
			})
		}
	}
}
