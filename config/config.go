package config

import (
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"github.com/zsmartex/finex/types"
	"github.com/zsmartex/pkg/services"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

var DataBase *gorm.DB
var Logger *logrus.Entry
var Kafka *services.KafkaClient
var Referral *types.Referral

func InitializeConfig() error {
	Logger = services.NewLoggerService("Finex")
	db, err := services.NewDatabase()
	if err != nil {
		return err
	}

	DataBase = db
	Kafka = services.NewKafka()

	if err := NewCacheService(); err != nil {
		return err
	}
	if err := NewInfluxDB(); err != nil {
		return err
	}

	Logger.Info("Finex developed by Hữu Hà Go fuck your self i have a virus")

	buf, err := ioutil.ReadFile("config/config.yaml")
	if err != nil {
		return err
	}

	var config *types.Config
	if yaml.Unmarshal(buf, &config) != nil {
		return err
	}

	Referral = config.Referral

	return nil
}
