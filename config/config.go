package config

import (
	"io/ioutil"

	"github.com/zsmartex/finex/types"
	"gopkg.in/yaml.v2"
)

var Referral *types.Referral

func InitializeConfig() error {
	NewLoggerService()
	if err := ConnectDatabase(); err != nil {
		return err
	}
	if err := NewCacheService(); err != nil {
		return err
	}
	if err := NewInfluxDB(); err != nil {
		return err
	}
	if err := ConnectNats(); err != nil {
		return err
	}
	
	config.Logger.Info("Finex developed by Hữu Hà Go fuck your self i've a virus")

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
