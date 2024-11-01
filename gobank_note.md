## Section 1: Working with database

### 1. Design DB schema and generate SQL code

Table `accounts`: 

Table `entries`: record all changes to the account balance.

Table `transfers`: records all the money transfers between 2 accounts

[dbDiagram]: https://dbdiagram.io/d/simplebank-670e279197a66db9a3036bd7

```postgresql
# Postgre
CREATE TABLE "accounts" (
  "id" bigserial PRIMARY KEY,
  "owner" varchar NOT NULL,
  "balance" bigint NOT NULL,
  "currency" varchar NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "entries" (
  "id" bigserial PRIMARY KEY,
  "account_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "transfers" (
  "id" bigserial PRIMARY KEY,
  "from_account_id" bigint NOT NULL,
  "to_account_id" bigint NOT NULL,
  "amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

ALTER TABLE "entries" ADD FOREIGN KEY ("account_id") REFERENCES "accounts" ("id");

ALTER TABLE "transfers" ADD FOREIGN KEY ("from_account_id") REFERENCES "accounts" ("id");

ALTER TABLE "transfers" ADD FOREIGN KEY ("to_account_id") REFERENCES "accounts" ("id");

CREATE INDEX ON "accounts" ("owner");

CREATE INDEX ON "entries" ("account_id");

CREATE INDEX ON "transfers" ("from_account_id");

CREATE INDEX ON "transfers" ("to_account_id");

CREATE INDEX ON "transfers" ("from_account_id", "to_account_id");

COMMENT ON COLUMN "entries"."amount" IS 'can be negative or positive';

COMMENT ON COLUMN "transfers"."amount" IS 'must be positive';
```



### 4. Docker + Postgres + TablePlus to create DB scheme

```shell
# Pull an image
docker pull <image>:<tag>
docker pull postgres:12-alpine

# Start a container
docker run --name <container_name> -e <environment_variable> -d <image>:<tag>
docker run --name postgres12 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine
```

Docker image & container
a container is 1 instance of the application contained in the image

```shell
# Port mapping `-p <host_ports:container_ports>`
docker run --name <container_name> -e <environment_variable> -p <host_ports:container_ports> -d <image>:<tag>

# Run command in container
docker exec -it <container_name_or_id> <command> [args]
docker exec -it postgres12 psql -U root
```

> Usage: docker exec [OPTIONS] CONTAINER COMMAND [ARG...]
>
> Execute a command in a running container
>
> -u, --user string     Username or UID

```shell
# View container logs
docker logs <container_name_or_id>
```



Docker common commands:

```shell
$ docker --help
```

> Common Commands:
>   run	 Create and run a new container from an image
>   exec       Execute a command in a running container
>   ps           List containers
>   build      Build an image from a Dockerfile
>   pull        Download an image from a registry
>   push      Upload an image to a registry
>   images  List images



### 5. database migration

`brew install golang-migrate`

**Up/Down migration**

Up: make a forward change to the schema 		OLD => NEW

Down: revert the change, rollback				NEW => OLD



**Check postgres container status**

```shell
docker ps

docker stop postgres12

# list all containers regardless of their running status
docker ps -a 

docker start postgres12

# shell: access postgres container shell (in ubuntu)
docker exec -it postgres12 /bin/sh

# without shell: access the database console without going through the container shell
docker exec -it postgres12 psql -U root simplebank
```



**Create/drop database**

```shell
# create/drop database inside postgres container
createdb --username=root --owner=root simple_bank
drop simple_bank

# create/drop database outside postgres container
docker exec -it postgres12 createdb --username=root --owner=root simple_bank 
```



**Migration**

```shell
# --- migrate up ---
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank" -verbose up
# Error: SSL is not enabled on the server. Because our postgres container doesn't enable SSL by default.

# sslmode=disable
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up

# --- migrate down ---
migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down
```



**Write Makefile**

```makefile
postgres:
    docker run --name postgres12 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine

createdb:
    docker exec -it postgres12 createdb --username=root --owner=root simple_bank

dropdb:
    docker exec -it postgres12 dropdb simple_bank

migrateup:
    migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up

migratedown:
    migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down

.PHONY: postgres createdb dropdb migrateup migratedown
```



### 6. Generate CRUD Golang code from SQL

sqlc SQL => Go

[Sqlc docs]: https://docs.sqlc.dev/en/latest/



**install sqlc** 

```shell
brew install kyleconroy/sqlc/sqlc
```



**write a setting file**

```shell
sqlc init
```

`sqlc.yaml`

```yaml
version: "1"
packages:
  - name: "db"
    path: "./db/sqlc"
    queries: "./db/query/"
    schema: "./db/migration/"
    engine: "postgresql"
    emit_json_tags: true
    emit_prepared_queries: false
    emit_interface: false
    emit_exact_table_names: false
```

> - The `name` option here is to tell sqlc what is the name of the Go package that will be generated. I think `db` is a good package name.
> - Next, we have to specify the `path` to the folder to store the generated golang code files. I’m gonna create a new folder `sqlc` inside the `db` folder, and change this `path` string to `./db/sqlc`.
> - Then we have the `queries` option to tell sqlc where to look for the SQL query files. Let’s create a new folder `query` inside the `db` folder. Then change this value to `./db/query`.
> - Similarly, this schema option should point to the folder containing the database schema or migration files. In our case, it is `./db/migration`.
> - The next option is `engine` to tell sqlc what database engine we would like to use. We’re using `Postgresql` for our simple bank project. If you want to experiment with MySQL, you can change this value to `mysql` instead.
> - Here we set the `emit_json_tags` to `true` because we want sqlc to add JSON tags to the generated structs.
> - The `emit_prepared_queries` tells sqlc to generate codes that work with prepared statement. At the moment, we don’t need to optimize performance yet, so let’s set this to `false` to make it simple.
> - Then the `emit_interface` option to tell sqlc to generate `Querier` interface for the generated package. It might be useful later if we want to mock the db for testing higher-level functions. For now let’s just set it to `false`.
> - The final option is `emit_exact_table_names`. By default, this value is `false`. Sqlc will try to singularize the table name to use as the model struct name. For example `accounts` table will become `Account` struct. If you set this option to true, the struct name will be `Accounts` instead. I think singular name is better because 1 object of type `Accounts` in plural form might be confused as multiple objects.

**Run sqlc generate command**

```makefile
...

sqlc:
    sqlc generate

.PHONY: postgres createdb dropdb migrateup migratedown sqlc
```

Create operation:





Go

[Pointer Receiver]: https://go.dev/tour/methods/4

the **receiver type** has the literal syntax **`*T`** for some type `T`. (Also, `T` cannot itself be a pointer such as `*int`.)

Methods with pointer receivers can **<u>modify the value</u> to which the receiver points**





7.

https://pkg.go.dev/testing



lib/bq

stretchr/testify



Main_test.go



Account_test.go



random.go

UnixNano()

Int63n





### 8. Implement db transaction in Golang

Composition

```go
type Store struct {
    *Queries // Composition: Embeds Queries struct for single-table query operations
    db       *sql.DB // Database connection used to manage transactions and connection pool
}
```





### 9. DB transaction lock & Handle deadlock

Query with Lock

```sql
SELECT * FROM accounts WHERE id = 1 FOR UPDATE;
```

它会被阻塞，必须等待第一个事务提交（`COMMIT`）或回滚（`ROLLBACK`）。通过添加 `FOR UPDATE` 子句，SQL 查询会锁定相关记录，确保其他并发事务在更新操作完成前无法访问这些记录。这种方式保证了数据一致性，避免了并发事务导致的更新冲突或不一致问题。



Deadlock

死锁问题的原因

死锁问题通常发生在两个或多个事务试图以不同的顺序锁定相同的资源，导致它们互相等待，无法继续执行。例如，在两个并发的转账操作中：

1. **事务 A**：尝试从账户 1 转出资金，并锁定了账户 1，然后试图锁定账户 2。
2. **事务 B**：尝试从账户 2 转出资金，并锁定了账户 2，然后试图锁定账户 1。

由于两个事务互相等待对方释放锁，因此形成了死锁。

如何解决死锁问题

要解决这个问题，通常有两种方法：

1. **强制锁定顺序**：确保所有并发的事务都按照相同的顺序获取锁定。比如，始终先锁定账户 ID 较小的账户，再锁定账户 ID 较大的账户。这样可以避免两个事务相互等待。
2. **设置事务重试机制**：如果检测到死锁，可以让事务重试操作。数据库系统有时能够自动检测并中止一个事务，让另一个事务完成。我们可以在应用层进行捕获，重新尝试执行被中止的事务。



### 10. avoid deadlock in DB transaction: Queries order



### 11 Transaction isolation levels & read phenomena

ACID Property





**4 Read Phenomena**

-  `dirty read` phenomenon. It happens when a transaction reads data written by <u>other concurrent transaction that has not been committed yet</u>. This is terribly bad, because we don’t know if that other transaction will eventually be committed or rolled back. So we might end up using incorrect data in case rollback occurs.
-  `non-repeatable read`. When a transaction <u>reads the same record twice and see different values</u>, because the *row has been modified by other transaction* that was committed after the first read.
- `Phantom read` is a similar phenomenon, but affects queries that search for <u>multiple rows</u> instead of one. In this case, <u>the same query is re-executed, but a different set of rows is returned</u>, due to some changes made by other recently-committed transactions, such as inserting new rows or deleting existing rows which happen to satisfy the search condition of current transaction’s query.
-  `serialization anomaly`. It’s when the result of a group of concurrent committed transactions could not be achieved if we try to run them sequentially <u>in any order</u> without overlapping each other.



**4 isolation levels**

-  `read uncommitted`. Transactions in this level can <u>see data written by other uncommitted transactions</u>, thus allowing `dirty read` phenomenon to happen.
-  `read committed`, where transactions can <u>only see data that has been committed by other transactions</u>. Because of this, `dirty read` is no longer possible.
-  `repeatable read` isolation level. It ensures that the <u>same select query will always return the same result</u>, no matter how many times it is executed, even if some other concurrent transactions have committed new changes that satisfy the query. `phantom read` phenomenon is also prevented in this `repeatable-read` isolation level
- `serializable`. Concurrent transactions running in this level are guaranteed to be able to yield the same result as if they’re executed sequentially in some order, one after another without overlapping. So basically it means that there exists at least 1 way to order these concurrent transactions so that if we run them one by one, the final result will be the same.

​	(transaction 1, transaction 2; if t1 accounts1.balance -1, the update query will be blocked because  t2 is blocking this update query in t1)



### 12 Github Actions for Golang + Postgres to run automated tests



**Workflow**

Workflow is basically an automated procedure that’s made up of one or more jobs. It can be triggered by 3 different ways:

- By an event that happens on the Github repository
- By setting a repetitive schedule
- Or manually clicking on the run workflow button on the repository UI.



`.yml` to `.github/workflows`

**Runner**

A runner is simply a server that listens for available jobs, and it will run only 1 job at a time.

The runners will run the jobs, then report the their progress, logs, and results back to Github, so we can easily check it on the UI of the repository.

```yml
jobs:
  build:
    runs-on: ubuntu-latest
```

----------

**Job**

A job is a set of steps that will be executed on the same runner.



The jobs are listed inside the workflow under the `jobs` keyword.

```yml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Check out code
        uses: actions/checkout@v2
      - name: Build server
        run: ./build_server.sh
  test:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - run: ./test_server.sh
```

-----------

**Steps**



```
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
```







# Section2: Building RESTful HTTP JSON API

### 13. RESTful API using Gin



### 14. Loading config with Viper

https://github.com/uber-go/mock



16.

Aud usd

20. stronger unit tests with gomock



weak unit test without gomock

```go
func TestCreateUserAPI(t *testing.T) {
    user, password := randomUser(t)

    testCases := []struct {
        name          string
        body          gin.H
        buildStubs    func(store *mockdb.MockStore)
        checkResponse func(recoder *httptest.ResponseRecorder)
    }{
        {
            name: "OK",
            body: gin.H{
                "username":  user.Username,
                "password":  password,
                "full_name": user.FullName,
                "email":     user.Email,
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Times(1).
                    Return(user, nil)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusOK, recorder.Code)
                requireBodyMatchUser(t, recorder.Body, user)
            },
        },
        {
            name: "InternalError",
            body: gin.H{
                "username":  user.Username,
                "password":  password,
                "full_name": user.FullName,
                "email":     user.Email,
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Times(1).
                    Return(db.User{}, sql.ErrConnDone)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusInternalServerError, recorder.Code)
            },
        },
        {
            name: "DuplicateUsername",
            body: gin.H{
                "username":  user.Username,
                "password":  password,
                "full_name": user.FullName,
                "email":     user.Email,
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Times(1).
                    Return(db.User{}, &pq.Error{Code: "23505"})
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusForbidden, recorder.Code)
            },
        },
        {
            name: "InvalidUsername",
            body: gin.H{
                "username":  "invalid-user#1",
                "password":  password,
                "full_name": user.FullName,
                "email":     user.Email,
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
        {
            name: "InvalidEmail",
            body: gin.H{
                "username":  user.Username,
                "password":  password,
                "full_name": user.FullName,
                "email":     "invalid-email",
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
        {
            name: "TooShortPassword",
            body: gin.H{
                "username":  user.Username,
                "password":  "123",
                "full_name": user.FullName,
                "email":     user.Email,
            },
            buildStubs: func(store *mockdb.MockStore) {
                store.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Times(0)
            },
            checkResponse: func(recorder *httptest.ResponseRecorder) {
                require.Equal(t, http.StatusBadRequest, recorder.Code)
            },
        },
    }

    for i := range testCases {
        tc := testCases[i]

        t.Run(tc.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            store := mockdb.NewMockStore(ctrl)
            tc.buildStubs(store)

            server := NewServer(store)
            recorder := httptest.NewRecorder()

            // Marshal body data to JSON
            data, err := json.Marshal(tc.body)
            require.NoError(t, err)

            url := "/users"
            request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
            require.NoError(t, err)

            server.router.ServeHTTP(recorder, request)
            tc.checkResponse(recorder)
        })
    }
}


```







# Section 3: Deploying the app to production (docker + kubernetes + aws)



how to build a minimal docker golang container

```dockerfile
# Build stage
FROM golang:1.23-alpine3.20 AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go

# Run stage
FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/main .

EXPOSE 8080
CMD [ "/app/main" ]
```



```shell
docker images
docker rmi f312ac8a88e6
docker rm simplebank
```



Use docker network to connect 2 stand-alone containers

```shell
docker run --name simplebank -p 8080:8080 -e GIN_MODE=release simplebank:latest 
docker ps
docker container inspect postgres12

docker network create bank-network

docker network connect bank-network postgres12

docker run --name simplebank --network bank-network -p 8080:8080 -e GIN_MODE=release -e DB_SOURCE="postgresql://root:secret@postgres12:5432/simple_bank?sslmode=disable" simplebank:latest


# update our docker image to run db migration before starting the api server

```



docker compose

https://docs.docker.com/reference/compose-file/





# Helpful Links

[Go in Visual Studio Code]: https://code.visualstudio.com/docs/languages/go



# A Tour of Go

