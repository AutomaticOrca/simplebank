Section 1

Use docker, postgres, tablepuls to create dbschema

1-8

DB transaction:

transfer record 10
Entry record A1 -10
Entry record A2 10
update A1 balance -10
Update A2 balance +10

BEGIN, COMMIT, ROLLBACK

Go:

`Store` struct: extend Queries's functionality

`Queries` 是由工具（如 `sqlc`）生成的一个结构体，包含了应用程序中与数据库交互的具体 SQL 查询方法。每个方法通常对应一个 SQL 查询或命令（如 `SELECT`、`INSERT`、`UPDATE`、`DELETE`）。

`sql.DB` 对象在整个应用程序的生命周期中通常只创建一次，然后在应用程序中被多次使用。它支持并发操作，并提供了自动管理连接的方法，如连接复用、连接超时等。

`&Store{}` is an expression used to get the address of the struct, which returns a pointer to the `Store` struct.



what is context

Context**上下文**是 Go 语言中用于在不同 goroutine 之间传递请求范围内的数据、取消信号和超时时间的一个机制。上下文通常用于控制请求的生命周期，例如设置超时时间或者在请求完成后取消它。

在数据库操作中使用上下文的主要原因有：

- **取消请求**：如果某个操作超时或被取消（比如用户取消了一个操作），上下文可以发出信号来中止正在进行的数据库操作，防止程序卡住。
- **设定超时时间**：可以在上下文中设置一个超时时间，让数据库操作在规定时间内完成，否则就会被取消。



what is Callback function

**回调函数**是指一个作为参数传递给另一个函数的函数。这个回调函数会在某个特定的时刻被调用。简单来说，回调函数就是你提供给某个操作的一个函数，让这个操作在合适的时候去执行你提供的函数。

在这个例子中，`execTx` 函数接受一个回调函数 `fn` 作为参数，这个函数 `fn` 包含了在事务中执行的所有操作。例如，它可能包含对数据库的插入、更新或删除等一系列操作。在 `execTx` 函数内部，这个回调函数会被调用，事务会根据这个回调函数的执行结果来决定是提交（`commit`）还是回滚（`rollback`）。



why make a new Queries object?

`Queries` 是一个包含一系列数据库操作的对象，在这里由 `sqlc` 生成。创建一个新的 `Queries` 对象并将其与事务相关联是为了让它在事务的上下文中执行查询操作。

在一个数据库事务中，所有的查询都必须在同一个事务对象中执行。以下是原因：

- **事务一致性**：通过使用 `Queries` 和事务对象，你可以确保事务中的所有操作（如插入、更新等）要么全部成功，要么全部失败，这就是事务的原子性。
- **隔离性**：事务中的操作在事务完成之前，对其他数据库操作是不可见的。将新的 `Queries` 与事务对象绑定，可以确保所有的数据库操作都在同一个事务中执行。

当你调用 `New(tx)` 时，实际上是创建了一个与这个特定事务关联的 `Queries` 对象，这样你在回调函数中执行的查询就都在同一个事务中进行了。



*execTx* function comment by chatGPT

```go
// execTx executes a function within a database transaction.
// It starts a new transaction, executes the provided function `fn` with the transaction context,
// and either commits the transaction if successful or rolls it back if an error occurs.
func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
    // Begin a new transaction using the database connection in `store`.
    // `ctx` is the context for transaction control (e.g., timeout, cancellation).
    // `&sql.TxOptions{}` specifies transaction options (here it's empty, using defaults).
  // 通过 store 的数据库连接启动一个新的事务。ctx 用于控制事务的上下文，&sql.TxOptions{} 用于指定事务选项。
    tx, err := store.db.BeginTx(ctx, &sql.TxOptions{})
    if err != nil {
        // If starting the transaction fails, return the error.
        return err
    }

    // Create a new `Queries` instance using the transaction. 创建一个新的 Queries 实例，它将在当前事务的上下文中执行数据库操作。
    // `q` will be used to perform database queries within this transaction.
    q := New(tx)

    // Execute the provided function `fn` with `q`. 使用 q 执行传入的函数 fn，它包含了具体的数据库操作。
    // This function should contain the database operations to be executed within the transaction.
    err = fn(q)
    if err != nil {
        // If the function `fn` returns an error, the transaction needs to be rolled back.
        // Attempt to rollback the transaction. 
        // rbErr := tx.Rollback() 这是一个简单的语句，用于在 if 语句的条件检查之前执行。
        if rbErr := tx.Rollback(); rbErr != nil {
            // If rollback fails, return both the original error and the rollback error.
            return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
        }
        // Return the original error from `fn`.
        return err
    }

    // If the function `fn` succeeded (err is nil), commit the transaction.
    // 如果 fn 成功执行，tx.Commit() 提交事务，持久化数据库更改。
    return tx.Commit()
}

```



那为什么func 后面的括号里得需要时一个 *Store类型呢, 为什么不能是Store呢?

如果你使用值接收者 `Store`，它会创建 `Store` 结构体的一个副本，并且在方法内部操作的将是这个副本，而不是原始的 `Store` 实例。这意味着你无法在方法中修改原始 `Store` 实例的状态（即使 `execTx` 方法不需要修改 `Store` 的状态，使用指针接收者仍然是一种惯例，因为它更高效且更具灵活性）。

**指针接收者允许修改接收者的状态**

当你使用指针接收者 `*Store` 时，你可以在方法中修改接收者的字段，而这些修改会在调用者中生效。虽然在 `execTx` 函数中没有直接修改 `Store` 的字段，但使用指针接收者仍然是一个常见的惯例，特别是对于涉及数据库操作的类型。因为：

- 指针接收者可以修改接收者内部的状态，比如更新数据库连接、缓存等。
- 如果你希望在方法中改变 `store` 的内部状态，指针接收者是必须的。





Why json:"from_account_id"?

it will transform struct to json! 
'encoding/json', json.Marshal,
"net/http", 

```go
import (
    "encoding/json"
    "net/http"
)

type TransferTxResult struct {
    TransferID int64 `json:"transfer_id"`
}

func transferHandler(w http.ResponseWriter, r *http.Request) {
    result := TransferTxResult{
        TransferID: 789,
    }

    // 设置响应头为 JSON
    w.Header().Set("Content-Type", "application/json")
    // 使用 json.NewEncoder 将结构体编码为 JSON 并写入到 http.ResponseWriter
    json.NewEncoder(w).Encode(result)
}

func main() {
    http.HandleFunc("/transfer", transferHandler)
    http.ListenAndServe(":8080", nil)
}

```







