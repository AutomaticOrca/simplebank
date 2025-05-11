-- name: CreateTransfer :one
INSERT INTO transfers (
  from_account_id,
  to_account_id,
  amount
) VALUES (
  $1, $2, $3
) RETURNING *;

-- name: GetTransfer :one
SELECT * FROM transfers
WHERE id = $1 LIMIT 1;

-- name: ListTransfers :many
SELECT * FROM transfers
WHERE 
    from_account_id = $1 OR
    to_account_id = $2
ORDER BY id
LIMIT $3
OFFSET $4;

-- name: ListTransfersByUsername :many
SELECT t.*
FROM transfers t
         JOIN accounts a_from ON t.from_account_id = a_from.id
         JOIN accounts a_to ON t.to_account_id = a_to.id
WHERE a_from.owner = $1 OR a_to.owner = $1
ORDER BY t.created_at DESC
LIMIT $2
    OFFSET $3;