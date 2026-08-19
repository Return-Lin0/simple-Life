// Package config 负责加载运行配置：
//   - 读取 config/config.yaml（非敏感项与密文）；
//   - 敏感项（AES 密钥、JWT 密钥）通过环境变量注入；
//   - 数据库密码运行时解密，任何缺失/解密失败都会导致启动失败（fail-fast）。
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	"vibe/internal/auth"
)

// Config 是全部运行配置的聚合结构。
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	MySQL    MySQLConfig    `mapstructure:"mysql"`
	Redis    RedisConfig    `mapstructure:"redis"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Security SecurityConfig `mapstructure:"security"`
	CORS     CORSConfig     `mapstructure:"cors"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	Reminder ReminderConfig `mapstructure:"reminder"`
}

// ServerConfig 服务监听配置。
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// MySQLConfig MySQL 连接配置（密码仅存密文）。
type MySQLConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	User           string `mapstructure:"user"`
	DBName         string `mapstructure:"dbname"`
	PasswordCipher string `mapstructure:"password_cipher"` // AES-256-GCM 密文
	MaxOpenConns   int    `mapstructure:"max_open_conns"`
	MaxIdleConns   int    `mapstructure:"max_idle_conns"`

	password string // 解密后的明文密码，仅存于进程内存，禁止日志输出
}

// Password 返回解密后的数据库密码；未解密时返回空串。
func (m *MySQLConfig) Password() string {
	return m.password
}

// RedisConfig Redis 连接配置。
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// RabbitMQConfig 消息队列连接配置。
type RabbitMQConfig struct {
	URL string `mapstructure:"url"`
}

// JWTConfig Token 相关配置。
type JWTConfig struct {
	SecretEnv       string `mapstructure:"secret_env"`
	AccessTTLMin    int    `mapstructure:"access_ttl_minutes"`
	RefreshTTLDays  int    `mapstructure:"refresh_ttl_days"`

	Secret       string        // 从环境变量读取的 JWT 密钥
	AccessTTL    time.Duration // 计算后的 Access Token 有效期
	RefreshTTL   time.Duration // 计算后的 Refresh Token 有效期
}

// SecurityConfig 安全策略配置。
type SecurityConfig struct {
	EnableLoginLock        bool `mapstructure:"enable_login_lock"` // FR-20，默认关闭待确认
	MaxLoginFailures       int  `mapstructure:"max_login_failures"`
	LockMinutes            int  `mapstructure:"lock_minutes"`
	APIRateLimitPerMinute  int  `mapstructure:"api_rate_limit_per_minute"`
}

// CORSConfig 跨域白名单。
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// LoggerConfig 日志级别。
type LoggerConfig struct {
	Level string `mapstructure:"level"`
}

// ReminderConfig 提醒调度参数。
type ReminderConfig struct {
	ScanIntervalSeconds int `mapstructure:"scan_interval_seconds"`
	WindowSeconds       int `mapstructure:"window_seconds"`
	CompensationBatch   int `mapstructure:"compensation_batch"`
}

// Load 加载配置并执行敏感项解析（环境变量、密码解密、TTL 计算）。
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath("config")
		v.AddConfigPath(".")
	}
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 环境变量覆盖：VIBE_ 前缀，点号转下划线（如 VIBE_MYSQL_HOST）
	v.SetEnvPrefix("VIBE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	if err := cfg.resolveSecrets(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// resolveSecrets 统一解析敏感配置项。
func (c *Config) resolveSecrets() error {
	// 1) 数据库密码解密：密钥必须来自环境变量 VIBE_DB_KEY
	if c.MySQL.PasswordCipher != "" {
		dbKey := os.Getenv("VIBE_DB_KEY")
		if dbKey == "" {
			return errors.New("配置了数据库密码密文，但缺少环境变量 VIBE_DB_KEY，拒绝启动")
		}
		plain, err := auth.DecryptAESGCM(dbKey, c.MySQL.PasswordCipher)
		if err != nil {
			return fmt.Errorf("数据库密码解密失败: %w", err)
		}
		c.MySQL.password = plain
	}

	// 2) JWT 密钥：必须来自环境变量，长度不足直接拒绝
	secret := os.Getenv(c.JWT.SecretEnv)
	if secret == "" {
		secret = os.Getenv("VIBE_JWT_SECRET")
	}
	if len(secret) < 32 {
		return errors.New("缺少 JWT 密钥（VIBE_JWT_SECRET ≥ 32 字节），拒绝启动")
	}
	c.JWT.Secret = secret

	// 3) 计算 Token 有效期
	c.JWT.AccessTTL = time.Duration(c.JWT.AccessTTLMin) * time.Minute
	if c.JWT.AccessTTL <= 0 {
		c.JWT.AccessTTL = 15 * time.Minute
	}
	c.JWT.RefreshTTL = time.Duration(c.JWT.RefreshTTLDays) * 24 * time.Hour
	if c.JWT.RefreshTTL <= 0 {
		c.JWT.RefreshTTL = 7 * 24 * time.Hour
	}

	// 4) 兜底默认值
	if c.Reminder.ScanIntervalSeconds <= 0 {
		c.Reminder.ScanIntervalSeconds = 60
	}
	if c.Reminder.WindowSeconds <= 0 {
		c.Reminder.WindowSeconds = 60
	}
	if c.Security.MaxLoginFailures <= 0 {
		c.Security.MaxLoginFailures = 5
	}
	if c.Security.LockMinutes <= 0 {
		c.Security.LockMinutes = 15
	}
	return nil
}
