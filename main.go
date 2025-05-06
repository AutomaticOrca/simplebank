package main

import (
	"crypto/tls"
	"database/sql"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/AutomaticOrca/simplebank/api"
	db "github.com/AutomaticOrca/simplebank/db/sqlc"
	"github.com/AutomaticOrca/simplebank/mail"
	"github.com/AutomaticOrca/simplebank/util"
	"github.com/AutomaticOrca/simplebank/worker"
	"github.com/hibiken/asynq"
	_ "github.com/lib/pq"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config:")
	}

	// main.go 中 LoadConfig 之后
	log.Info().
		Str("db_driver", config.DBDriver).
		Str("db_source_is_empty", fmt.Sprintf("%t", config.DBSource == "")).
		Str("http_server_address", config.HTTPServerAddress).
		Str("redis_address", config.RedisAddress).
		Str("redis_password_is_empty", fmt.Sprintf("%t", config.RedisPassword == "")). // 检查 Redis 密码
		// 打印所有其他你关心的配置项...
		Msg("Loaded configuration values from environment")

	if config.DBDriver == "" {
		log.Fatal().Msg("DB_DRIVER configuration is empty, cannot proceed")
	}
	if config.RedisAddress == "" { // 示例：添加对 Redis 地址的检查
		log.Fatal().Msg("REDIS_ADDRESS configuration is empty, cannot proceed")
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db")
	}

	store := db.NewStore(conn)

	redisOpt := asynq.RedisClientOpt{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
		Username: "default",
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	taskDistributor := worker.NewRedisTaskDistributor(redisOpt)
	go runTaskProcessor(config, redisOpt, store)

	server, err := api.NewServer(config, store, taskDistributor)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create server")
	}

	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start server")
	}
}

func runTaskProcessor(config util.Config, redisOpt asynq.RedisClientOpt, store db.Store) {
	mailer := mail.NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)
	taskProcessor := worker.NewRedisTaskProcessor(redisOpt, store, mailer)
	log.Info().Msg("start task processor")
	err := taskProcessor.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start task processor")
	}
}
