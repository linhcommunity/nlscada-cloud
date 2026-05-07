package influxdb

import (
	"context"
	"fmt"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type Reader struct {
	client   influxdb2.Client
	queryAPI api.QueryAPI
}

func NewReader(url, token, org string) (*Reader, error) {
	client := influxdb2.NewClient(url, token)
	// Kiểm tra kết nối
	if _, err := client.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping InfluxDB: %w", err)
	}
	return &Reader{
		client:   client,
		queryAPI: client.QueryAPI(org),
	}, nil
}

func (r *Reader) Query(query string) (*api.QueryTableResult, error) {
	return r.queryAPI.Query(context.Background(), query)
}

func (r *Reader) Close() {
	r.client.Close()
}
