package util

import (
	"errors" // 用于创建自定义错误
	"fmt"    // 用于格式化错误消息
	"os"
	"time"

	"github.com/rs/zerolog/log"

	// 这会自动加载你的 .env 文件 (通常是项目根目录下的 .env)
	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	Environment          string
	DBDriver             string
	DBSource             string
	RedisAddress         string
	RedisPassword        string // 如果 Redis 不需要密码，此项可以为空
	HTTPServerAddress    string
	TokenSymmetricKey    string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	EmailSenderName      string
	EmailSenderAddress   string
	EmailSenderPassword  string
}

func LoadConfig() (cfg Config, err error) {
	log.Info().Msg("Loading configuration directly from environment variables...")
	cfg.Environment = os.Getenv("ENVIRONMENT")
	cfg.DBDriver = os.Getenv("DB_DRIVER")
	cfg.DBSource = os.Getenv("DB_SOURCE")
	cfg.RedisAddress = os.Getenv("REDIS_ADDRESS")
	cfg.RedisPassword = os.Getenv("REDIS_PASSWORD") // 可选
	cfg.HTTPServerAddress = os.Getenv("HTTP_SERVER_ADDRESS")
	cfg.TokenSymmetricKey = os.Getenv("TOKEN_SYMMETRIC_KEY")
	cfg.EmailSenderName = os.Getenv("EMAIL_SENDER_NAME")
	cfg.EmailSenderAddress = os.Getenv("EMAIL_SENDER_ADDRESS")
	cfg.EmailSenderPassword = os.Getenv("EMAIL_SENDER_PASSWORD")

	// --- 必要配置项检查 (示例) ---
	// 你可以根据需要，对认为必须存在的配置项进行检查
	if cfg.DBDriver == "" {
		return Config{}, errors.New("DB_DRIVER environment variable is not set or is empty")
	}
	if cfg.DBSource == "" {
		return Config{}, errors.New("DB_SOURCE environment variable is not set or is empty")
	}
	if cfg.HTTPServerAddress == "" {
		return Config{}, errors.New("HTTP_SERVER_ADDRESS environment variable is not set or is empty")
	}
	if cfg.TokenSymmetricKey == "" {
		return Config{}, errors.New("TOKEN_SYMMETRIC_KEY environment variable is not set or is empty")
	}
	// ... 其他必要配置项的检查

	// 读取并解析 time.Duration 类型的环境变量
	accessTokenDurationStr := os.Getenv("ACCESS_TOKEN_DURATION")
	if accessTokenDurationStr == "" {
		return Config{}, errors.New("ACCESS_TOKEN_DURATION environment variable is not set or is empty")
	}
	cfg.AccessTokenDuration, err = time.ParseDuration(accessTokenDurationStr)
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse ACCESS_TOKEN_DURATION: %w", err)
	}

	refreshTokenDurationStr := os.Getenv("REFRESH_TOKEN_DURATION")
	if refreshTokenDurationStr == "" {
		return Config{}, errors.New("REFRESH_TOKEN_DURATION environment variable is not set or is empty")
	}
	cfg.RefreshTokenDuration, err = time.ParseDuration(refreshTokenDurationStr)
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse REFRESH_TOKEN_DURATION: %w", err)
	}

	// --- 电子邮件相关配置检查 (示例，如果邮件功能是核心功能) ---
	if cfg.EmailSenderAddress != "" { // 如果设置了发送地址，则认为邮件功能被启用
		if cfg.EmailSenderName == "" {
			return Config{}, errors.New("EMAIL_SENDER_NAME must be set if EMAIL_SENDER_ADDRESS is set")
		}
		if cfg.EmailSenderPassword == "" {
			return Config{}, errors.New("EMAIL_SENDER_PASSWORD must be set if EMAIL_SENDER_ADDRESS is set")
		}
	}

	return cfg, nil
}
