package util

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Environment          string        `mapstructure:"ENVIRONMENT"`
	DBDriver             string        `mapstructure:"DB_DRIVER"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	RedisAddress         string        `mapstructure:"REDIS_ADDRESS"`
	RedisPassword        string        `mapstructure:"REDIS_PASSWORD"`
	HTTPServerAddress    string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	EmailSenderName      string        `mapstructure:"EMAIL_SENDER_NAME"`
	EmailSenderAddress   string        `mapstructure:"EMAIL_SENDER_ADDRESS"`
	EmailSenderPassword  string        `mapstructure:"EMAIL_SENDER_PASSWORD"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	// 尝试读取文件，但主要依赖环境变量
	_ = viper.ReadInConfig() // 让它尝试读，但我们不依赖它

	err = viper.BindEnv("DBDriver", "DB_DRIVER")
	if err != nil {
		return
	} // 简单处理错误，实际可以打日志
	err = viper.BindEnv("DBSource", "DB_SOURCE")
	if err != nil {
		return
	}
	err = viper.BindEnv("HTTPServerAddress", "HTTP_SERVER_ADDRESS")
	if err != nil {
		return
	}
	err = viper.BindEnv("RedisAddress", "REDIS_ADDRESS")
	if err != nil {
		return
	}
	err = viper.BindEnv("RedisPassword", "REDIS_PASSWORD")
	if err != nil {
		return
	}
	// 为其他所有需要的字段也添加类似的 BindEnv
	err = viper.BindEnv("TokenSymmetricKey", "TOKEN_SYMMETRIC_KEY")
	if err != nil {
		return
	}
	err = viper.BindEnv("AccessTokenDuration", "ACCESS_TOKEN_DURATION")
	if err != nil {
		return
	}
	err = viper.BindEnv("RefreshTokenDuration", "REFRESH_TOKEN_DURATION")
	if err != nil {
		return
	}
	err = viper.BindEnv("EmailSenderName", "EMAIL_SENDER_NAME")
	if err != nil {
		return
	}
	err = viper.BindEnv("EmailSenderAddress", "EMAIL_SENDER_ADDRESS")
	if err != nil {
		return
	}
	err = viper.BindEnv("EmailSenderPassword", "EMAIL_SENDER_PASSWORD")
	if err != nil {
		return
	}

	viper.AutomaticEnv() // 仍然调用它，以覆盖其他未显式绑定的字段

	err = viper.Unmarshal(&config)
	return
}
