package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr       string
	WorkerCount    int
	QueueSize      int
	JobTimeout     time.Duration
	ShutdownTimout time.Duration
	Env            string
}

func Load() Config {
	return Config{
		HTTPAddr:       getEnv("HTTP_ADDR", ":8080"),
		WorkerCount:    getInt("WORKER_COUNT", 4),
		QueueSize:      getInt("QUEUE_SIZE", 64),
		JobTimeout:     getDuration("JOB_TIMEOUT", 15*time.Second),
		ShutdownTimout: getDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		Env:            getEnv("APP_ENV", "development"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			return parsed
		}
	}
	return def
}

