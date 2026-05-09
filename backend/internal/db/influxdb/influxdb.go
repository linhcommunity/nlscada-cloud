package influxdb

import (
	"context"
	"fmt"
	"log"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type Writer struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
	bucket   string
	org      string
}

func NewWriter(url, token, org, bucket string) (*Writer, error) {
	client := influxdb2.NewClient(url, token)
	writeAPI := client.WriteAPIBlocking(org, bucket)

	// Kiểm tra kết nối
	_, err := client.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to ping InfluxDB: %w", err)
	}

	log.Println("InfluxDB connected")
	return &Writer{
		client:   client,
		writeAPI: writeAPI,
		bucket:   bucket,
		org:      org,
	}, nil
}

// WritePoint ghi một point vào InfluxDB
func (w *Writer) WritePoint(measurement string, tags map[string]string, fields map[string]interface{}, ts time.Time) error {
	p := influxdb2.NewPoint(measurement, tags, fields, ts)
	return w.writeAPI.WritePoint(context.Background(), p)
}

func (w *Writer) Close() {
	w.client.Close()
	log.Println("InfluxDB disconnected")
}
