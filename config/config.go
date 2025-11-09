package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Redis   RedisConfig   `mapstructure:"redis"`
	Upload  UploadConfig  `mapstructure:"upload"`
	GrabCut GrabCutConfig `mapstructure:"grabcut"`
}

type ServerConfig struct {
	Port         string        `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type RedisConfig struct {
	Addr     string        `mapstructure:"addr"`
	Password string        `mapstructure:"password"`
	DB       int           `mapstructure:"db"`
	TTL      time.Duration `mapstructure:"ttl"`
}

type UploadConfig struct {
	MaxSize      int64    `mapstructure:"max_size"`
	UploadDir    string   `mapstructure:"upload_dir"`
	AllowedTypes []string `mapstructure:"allowed_types"`
}

type GrabCutConfig struct {
	Iterations       int  `mapstructure:"iterations"`
	BorderSize       int  `mapstructure:"border_size"`
	MaxConcurrent    int  `mapstructure:"max_concurrent"`
	QueueTimeout     int  `mapstructure:"queue_timeout"`
	CleanupTempFiles bool `mapstructure:"cleanup_temp_files"`
}

// Load 从 YAML 文件加载配置
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 设置默认值
	setDefaults(v)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// New 使用默认配置路径加载配置
func New() *Config {
	cfg, err := Load("config.yaml")
	if err != nil {
		// 如果加载失败，返回默认配置
		return getDefaultConfig()
	}
	return cfg
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", ":8080")
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.read_timeout", 10*time.Second)
	v.SetDefault("server.write_timeout", 10*time.Second)

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.ttl", 24*time.Hour)

	v.SetDefault("upload.max_size", 10*1024*1024)
	v.SetDefault("upload.upload_dir", "./uploads")
	v.SetDefault("upload.allowed_types", []string{"image/jpeg", "image/png", "image/jpg"})

	v.SetDefault("grabcut.iterations", 5)
	v.SetDefault("grabcut.border_size", 10)
	v.SetDefault("grabcut.max_concurrent", 3)
	v.SetDefault("grabcut.queue_timeout", 60)
	v.SetDefault("grabcut.cleanup_temp_files", true)
	v.SetDefault("grabcut.max_concurrent", 3)
	v.SetDefault("grabcut.queue_timeout", 30)
	v.SetDefault("grabcut.cleanup_temp_files", true)
}

func getDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         ":8080",
			Mode:         "debug",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
			TTL:      24 * time.Hour,
		},
		Upload: UploadConfig{
			MaxSize:      10 * 1024 * 1024,
			UploadDir:    "./uploads",
			AllowedTypes: []string{"image/jpeg", "image/png", "image/jpg"},
		},
		GrabCut: GrabCutConfig{
			Iterations:       5,
			BorderSize:       10,
			MaxConcurrent:    3,
			QueueTimeout:     30,
			CleanupTempFiles: true,
		},
	}
}
