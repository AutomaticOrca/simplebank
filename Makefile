-include .env

DB_SSL_MODE ?= disable
DB_SOURCE="postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSL_MODE}"

postgres:
	docker run --name postgres12 -p ${DB_PORT}:${DB_PORT} -e POSTGRES_USER=${DB_USER} -e POSTGRES_PASSWORD=${DB_PASSWORD} -d postgres:12-alpine

createdb:
	docker exec -it postgres12 createdb --username=${DB_USER} --owner=${DB_USER} ${DB_NAME}

dropdb:
	docker exec -it postgres12 dropdb --username=${DB_USER} ${DB_NAME}

migrateup:
	migrate -path db/migration -database "${DB_SOURCE}" -verbose up

migrateup1:
	migrate -path db/migration -database "${DB_SOURCE}" -verbose up 1

migratedown:
	migrate -path db/migration -database "${DB_SOURCE}" -verbose down

migratedown1:
	migrate -path db/migration -database "${DB_SOURCE}" -verbose down 1

new_migration:
	migrate create -ext sql -dir db/migration -seq $(name)

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

server:
	go run main.go

redis:
	docker run --name redis -p 6379:6379 -d redis:7-alpine

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/AutomaticOrca/simplebank/db/sqlc Store

lint:
	golangci-lint run


.PHONY: postgres createdb dropdb migrateup migratedown migrateup1 migratedown1 new_migration sqlc test server mock lint lint-install
