package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter          metric.Meter
	requestCounter metric.Int64Counter
	requestLatency metric.Float64Histogram
	serviceName    string
)

// SetMeter 从外部设置 meter
func SetMeter(m metric.Meter, name string) error {
	meter = m
	serviceName = name

	var err error
	requestCounter, err = meter.Int64Counter(
		fmt.Sprintf("%s_rpc_requests_total", serviceName),
		metric.WithDescription("Total number of RPC requests"),
	)
	if err != nil {
		return err
	}

	requestLatency, err = meter.Float64Histogram(
		fmt.Sprintf("%s_rpc_request_duration_seconds", serviceName),
		metric.WithDescription("RPC request duration in seconds"),
	)
	if err != nil {
		return err
	}

	return nil
}

func RecordRequest(ctx context.Context, method, status string, duration float64) {
	if meter == nil {
		return
	}

	attributes := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.String("status", status),
	}

	requestCounter.Add(ctx, 1, metric.WithAttributes(attributes...))
	requestLatency.Record(ctx, duration, metric.WithAttributes(
		attribute.String("method", method),
	))
}
