package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config はアプリケーション全体の設定を表現します。
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
}

// ServerConfig は gRPC サーバーに関する設定です。
type ServerConfig struct {
	ListenAddr string `yaml:"listen_addr"`
}

// DatabaseConfig は PostgreSQL 接続に関する設定です。
type DatabaseConfig struct {
	Host               string        `yaml:"host"`
	Port               int           `yaml:"port"`
	User               string        `yaml:"user"`
	Password           string        `yaml:"password"`
	Name               string        `yaml:"name"`
	SSLMode            string        `yaml:"ssl_mode"`
	MaxOpenConns       int           `yaml:"max_open_conns"`
	MaxIdleConns       int           `yaml:"max_idle_conns"`
	ConnMaxLifetime    time.Duration `yaml:"-"`
	ConnMaxIdleTime    time.Duration `yaml:"-"`
	ConnMaxLifetimeRaw string        `yaml:"conn_max_lifetime"`
	ConnMaxIdleTimeRaw string        `yaml:"conn_max_idle_time"`
}

// Load は指定されたパスから設定ファイルを読み込みます。
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse yaml: %w", err)
	}

	if err := cfg.validateAndNormalize(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) validateAndNormalize() error {
	if c.Server.ListenAddr == "" {
		return fmt.Errorf("config: server.listen_addr must be set")
	}

	db := &c.Database
	if err := db.validateAndNormalize(); err != nil {
		return err
	}

	return nil
}

func (d *DatabaseConfig) validateAndNormalize() error {
	if d.Host == "" {
		return fmt.Errorf("config: database.host must be set")
	}
	if d.Port == 0 {
		return fmt.Errorf("config: database.port must be set")
	}
	if d.User == "" {
		return fmt.Errorf("config: database.user must be set")
	}
	if d.Password == "" {
		return fmt.Errorf("config: database.password must be set")
	}
	if d.Name == "" {
		return fmt.Errorf("config: database.name must be set")
	}
	if d.SSLMode == "" {
		d.SSLMode = "disable"
	}

	lifetime, err := parseDurationAllowEmpty(d.ConnMaxLifetimeRaw)
	if err != nil {
		return fmt.Errorf("config: database.conn_max_lifetime: %w", err)
	}
	d.ConnMaxLifetime = lifetime

	idleTime, err := parseDurationAllowEmpty(d.ConnMaxIdleTimeRaw)
	if err != nil {
		return fmt.Errorf("config: database.conn_max_idle_time: %w", err)
	}
	d.ConnMaxIdleTime = idleTime

	return nil
}

func parseDurationAllowEmpty(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, err
	}
	return d, nil
}

// DSN は pgx 用の接続文字列を返します。
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)
}
