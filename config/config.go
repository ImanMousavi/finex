package config

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

	return nil
}
