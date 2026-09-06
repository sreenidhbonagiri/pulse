package config

// Config holds app settings. More fields (database, Redis, etc.) will be added later.
type Config struct {
	Addr string
}

func Load() Config {
	return Config{
		Addr: ":8080",
	}
}
