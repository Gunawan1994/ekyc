package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App   AppConfig   `mapstructure:"app"`
	DB    DBConfig    `mapstructure:"db"`
	Redis RedisConfig `mapstructure:"redis"`
	JWT   JWTConfig   `mapstructure:"jwt"`
	Log   LogConfig   `mapstructure:"log"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
	MaxConns int    `mapstructure:"maxconns"`
	MinConns int    `mapstructure:"minconns"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret             string        `mapstructure:"secret"`
	AccessTokenExpiry  time.Duration `mapstructure:"accesstokenexpiry"`
	RefreshTokenExpiry time.Duration `mapstructure:"refreshtokenexpiry"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s pool_max_conns=%d pool_min_conns=%d",
		d.Host,
		d.Port,
		d.User,
		d.Password,
		d.Name,
		d.SSLMode,
		d.MaxConns,
		d.MinConns,
	)
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/app")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults()
	bindEnvs()

	_ = viper.ReadInConfig()

	cfg := &Config{}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func bindEnvs() {
	// APP
	_ = viper.BindEnv("app.name", "APP_NAME")
	_ = viper.BindEnv("app.env", "APP_ENV")
	_ = viper.BindEnv("app.port", "APP_PORT")
	_ = viper.BindEnv("app.host", "APP_HOST")

	// DB
	_ = viper.BindEnv("db.host", "DB_HOST")
	_ = viper.BindEnv("db.port", "DB_PORT")
	_ = viper.BindEnv("db.user", "DB_USER")
	_ = viper.BindEnv("db.password", "DB_PASSWORD")
	_ = viper.BindEnv("db.name", "DB_NAME")
	_ = viper.BindEnv("db.sslmode", "DB_SSLMODE")
	_ = viper.BindEnv("db.maxconns", "DB_MAXCONNS")
	_ = viper.BindEnv("db.minconns", "DB_MINCONNS")

	// Redis
	_ = viper.BindEnv("redis.host", "REDIS_HOST")
	_ = viper.BindEnv("redis.port", "REDIS_PORT")
	_ = viper.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = viper.BindEnv("redis.db", "REDIS_DB")

	// JWT
	_ = viper.BindEnv("jwt.secret", "JWT_SECRET")
	_ = viper.BindEnv("jwt.accesstokenexpiry", "JWT_ACCESSTOKENEXPIRY")
	_ = viper.BindEnv("jwt.refreshtokenexpiry", "JWT_REFRESHTOKENEXPIRY")

	// Log
	_ = viper.BindEnv("log.level", "LOG_LEVEL")
	_ = viper.BindEnv("log.format", "LOG_FORMAT")
}

func setDefaults() {
	viper.SetDefault("app.name", "ekyc-platform")
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.port", 8080)
	viper.SetDefault("app.host", "0.0.0.0")

	viper.SetDefault("db.host", "localhost")
	viper.SetDefault("db.port", 5432)
	viper.SetDefault("db.sslmode", "disable")
	viper.SetDefault("db.maxconns", 25)
	viper.SetDefault("db.minconns", 5)

	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)

	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
}

func validate(cfg *Config) error {
	if cfg.DB.User == "" {
		return fmt.Errorf("DB_USER is required")
	}

	if cfg.DB.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}

	if cfg.DB.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}

	if cfg.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	return nil
}
