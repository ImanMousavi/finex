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
var KafkaProducer *services.KafkaProducer
var RangoClient *services.RangoClient
var Referral *types.Referral

func InitializeConfig() error {
	Logger = services.NewLoggerService("Finex")
	db, err := NewDatabase()
	if err != nil {
		return err
	}

	DataBase = db
	KafkaProducer, err = services.NewKafkaProducer(Logger)
	if err != nil {
		return err
	}

	RangoClient, err = services.NewRangoClient(KafkaProducer)
	if err != nil {
		return err
	}

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
