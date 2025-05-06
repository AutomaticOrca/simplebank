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
	viper.AddConfigPath(path)  // 比如 "."
	viper.SetConfigName("app") // 会寻找 app.env, app.yaml 等
	viper.SetConfigType("env") // 明确指定文件类型为 env

	// 尝试读取配置文件。在生产环境中，如果文件不存在，我们更依赖环境变量。
	// _ = viper.ReadInConfig() // 我们之前讨论过，可以更明确地处理这个错误
	if errRead := viper.ReadInConfig(); errRead != nil {
		if _, ok := errRead.(viper.ConfigFileNotFoundError); ok {
			// log.Info().Msg("Configuration file 'app.env' not found. Relying on environment variables.") // 【建议】加入日志
		} else {
			// log.Warn().Err(errRead).Msg("Error reading configuration file.") // 【建议】加入日志
		}
	}

	// 让 viper 也能从环境变量中读取配置
	// 例如，DB_DRIVER 环境变量会自动映射到 Config 结构体的 DBDriver 字段
	// (前提是 mapstructure 标签匹配或字段名自动转换匹配，viper 默认会将环境变量名转为大写)
	viper.AutomaticEnv()

	err = viper.Unmarshal(&config)
	return
}
