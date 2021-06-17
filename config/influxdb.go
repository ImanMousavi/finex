package config

import (
	"os"
	"time"

	"github.com/influxdata/influxdb/client/v2"
)

var InfluxDB *InfluxClient

type InfluxClient struct {
	client client.Client
}

func NewInfluxDB() error {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: os.Getenv("INFLUXDB_URL"),
	})

	if err != nil {
		return err
	}

	InfluxDB = &InfluxClient{
		client: c,
	}

	return nil
}

func (c *InfluxClient) NewBatchPoints() (client.BatchPoints, error) {
	return client.NewBatchPoints(client.BatchPointsConfig{
		Database:  os.Getenv("INFLUXDB_DATABASE"),
		Precision: "ns",
	})
}

func (c *InfluxClient) NewPoint(name string, tags map[string]string, fields map[string]interface{}) {
	bp, err := c.NewBatchPoints()

	if err != nil {
		Logger.Errorf("Failed to create new batch point %v", err.Error())
	}

	point, err := client.NewPoint(name, tags, fields, time.Now())
	if err != nil {
		Logger.Errorf("Error %v", err.Error())
	}

	bp.AddPoint(point)

	// Write the batch
	err = c.client.Write(bp)
	if err != nil {
		Logger.Errorf("Error %v", err.Error())
	}
}

func (c *InfluxClient) Query() {
	// q := client.NewQuery("SELECT id, price, amount, total, taker_type, market, created_at FROM trades WHERE market=ethusdt", "square_holes", "ns")
}
