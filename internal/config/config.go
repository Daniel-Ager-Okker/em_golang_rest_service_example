package config

import (
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	DevEnv  = "dev"
	ProdEnv = "prod"
)

type Config struct {
	Env        string `yaml:"env"`
	StorageCfg `yaml:"storage"`
	HTTPServer `yaml:"http_server"`
}

type HTTPServer struct {
	Address     string        `yaml:"address"`
	Timeout     time.Duration `yaml:"timeout"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
}

type StorageCfg struct {
	// Dev env
	StoragePath string `yaml:"storage_path"`

	// Prod env
	PgHost               string        `yaml:"pg_host"`
	PgPort               int           `yaml:"pg_port"`
	PgDbName             string        `yaml:"pg_db_name"`
	PgMaxPoolSize        int           `yaml:"pg_max_pool_size"`
	PgConnectionAttempts int           `yaml:"pg_connection_attempts"`
	PgConnectionTimeout  time.Duration `yaml:"pg_connection_timeout"`
}

// Load configuration from YAML file
func Load() (*Config, error) {
	// 1.Get configuration file path from CONFIG_PATH env
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Println("CONFIG_PATH environment variable is not set")
		return nil, errors.New("CONFIG_PATH environment variable is not set")
	}

	// 2.Check if exists
	if _, err := os.Stat(configPath); err != nil {
		log.Printf("error opening config file: %s\n", err)
		return nil, err
	}

	// 3.Read
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("error reading config file: %s\n", err)
		return nil, err
	}

	// 4.Parse
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Printf("error parsing config file: %s\n", err)
		return nil, err
	}

	// 5.Validate configuration
	err = validateCfg(&cfg)
	if err != nil {
		log.Printf("configuration is invalid: %s\n", err)
		return nil, err
	}

	return &cfg, nil
}

// Validate configuration
func validateCfg(cfg *Config) error {
	// 1.HTTP params validation
	if strings.Compare(cfg.Address, "") == 0 {
		log.Println("key 'address' of tag 'http_server' not set, use default '0.0.0.0:8082'")
		cfg.Address = "0.0.0.0:8082"
	}

	if cfg.Timeout == 0 {
		log.Println("key 'timeout' of tag 'http_server' not set, use default '4s'")
		cfg.Timeout = 4 * time.Second
	}

	if cfg.IdleTimeout == 0 {
		log.Println("key 'idle_timeout' of tag 'http_server' not set, use default '30s'")
		cfg.Timeout = 30 * time.Second
	}

	// 2.Environment params validation
	if strings.Compare(cfg.Env, "") == 0 {
		return errors.New("must specify 'env' key in configuration")
	}

	switch cfg.Env {
	case DevEnv:
		return handleDevEnv(&cfg.StorageCfg)
	case ProdEnv:
		return handleProdEnv(&cfg.StorageCfg)
	default:
		return errors.New("unsupported 'env' value (use 'dev' or 'prod' only)")
	}
}

// Handle dev env params
func handleDevEnv(cfg *StorageCfg) error {
	if strings.Compare(cfg.StoragePath, "") == 0 {
		return errors.New("must specify 'storage_path' key while using 'dev' env")
	}
	return nil
}

// Handle prod validateProdEnv params
func handleProdEnv(cfg *StorageCfg) error {
	// 1.Required params
	if strings.Compare(cfg.PgHost, "") == 0 {
		return errors.New("must specify 'pg_host' key while using 'prod' env")
	}
	if cfg.PgPort == 0 {
		return errors.New("must specify 'pg_port' key while using 'prod' env")
	}
	if strings.Compare(cfg.PgDbName, "") == 0 {
		return errors.New("must specify 'pg_db_name' key while using 'prod' env")
	}

	// 2.Optional params
	if cfg.PgMaxPoolSize == 0 {
		log.Println("param 'pg_max_pool_size' is unset, so use default 1")
		cfg.PgMaxPoolSize = 1
	}

	if cfg.PgConnectionAttempts == 0 {
		log.Println("param 'pg_connection_attempts' is unset, so use default 3")
		cfg.PgConnectionAttempts = 3
	}

	if cfg.PgConnectionTimeout == 0 {
		log.Println("param 'pg_connection_timeout' is unset, so use default 30s")
		cfg.PgConnectionTimeout = 30 * time.Second
	}

	return nil
}
