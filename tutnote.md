## Section 1: Working with database

### 4. Use Docker + Postgres + TablePlus to create DB scheme

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



### 5. How to write & run database migration in Golang

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
docker exec -it postgres12 psql -U root simplebook
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



6. ### Generate CRUD Golang code from SQL | Compare db/sql, gorm, sqlx & sqlc

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





### 5. A clean way to implement db transaction in Golang

Every unit test function in Go must start with the `Test` prefix (with uppercase letter T) and takes a `testing.T` object as input.

store: provides all function to execute db queries and transactions
