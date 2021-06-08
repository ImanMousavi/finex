package config

import (
	"log"
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
		log.Println("Failed to create new batch point", err.Error())
	}

	point, err := client.NewPoint(name, tags, fields, time.Now())
	if err != nil {
		log.Println("Error: ", err.Error())
	}

	log.Println("Writing point to influxdb")

	bp.AddPoint(point)

	// Write the batch
	err = c.client.Write(bp)
	if err != nil {
		log.Println("Error: ", err.Error())
	}
}

func (c *InfluxClient) Query() {
	// q := client.NewQuery("SELECT id, price, amount, total, taker_type, market, created_at FROM trades WHERE market=ethusdt", "square_holes", "ns")
}
