package config

func InitializeConfig() error {
	if err := ConnectDatabase(); err != nil {
		return err
	}
	if err := NewCacheService(); err != nil {
		return err
	}
	if err := NewInfluxDB(); err != nil {
		return err
	}

	return nil
}
