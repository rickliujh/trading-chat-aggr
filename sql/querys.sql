-- name: ListBars :many
SELECT * FROM OHLC1M 
ORDER BY ts;

-- name: CreateBar :one
INSERT INTO OHLC1M (
  h, l, o, c, ts
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: UpdateBar :exec
UPDATE OHLC1M
  set h = $2,
 l = $3,
 o = $4,
 c = $5
WHERE id = $1;

-- name: DeleteBar :exec
DELETE FROM OHLC1M
WHERE id = $1;
