package config

func InitializeConfig() error {
	if err := NewCacheService(); err != nil {
		return err
	}
	if err := ConnectDatabase(); err != nil {
		return err
	}

	return nil
}
