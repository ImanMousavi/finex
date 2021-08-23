package config

import (
	"os"
	"time"

	"github.com/cbrake/influxdbhelper/v2"
	client "github.com/influxdata/influxdb1-client/v2"
)

var InfluxDB *InfluxClient

type InfluxClient struct {
	client influxdbhelper.Client
}

func NewInfluxDB() error {
	influx, err := influxdbhelper.NewClient(os.Getenv("INFLUXDB_URL"), "", "", "ns")
	if err != nil {
		return err
	}

	InfluxDB = &InfluxClient{
		client: influx.UseDB(os.Getenv("INFLUXDB_DATABASE")),
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
		return
	}

	point, err := client.NewPoint(name, tags, fields, time.Now())
	if err != nil {
		Logger.Errorf("Error %v", err.Error())
		return
	}

	bp.AddPoint(point)

	// Write the batch
	err = c.client.Write(bp)
	if err != nil {
		Logger.Errorf("Error %v", err.Error())
		return
	}
}

func (c *InfluxClient) Query(command string, result interface{}) error {
	return c.client.DecodeQuery(command, result)
}
