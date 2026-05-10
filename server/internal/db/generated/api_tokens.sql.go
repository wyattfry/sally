package generated

import (
	"context"
	"database/sql"
)

const createAPIToken = `
insert into api_tokens (user_id, label, token_hash)
values ($1, $2, $3)
returning id, user_id, label, token_hash, created_at, last_used_at
`

func (q *Queries) CreateAPIToken(ctx context.Context, userID, label, tokenHash string) (APIToken, error) {
	row := q.db.QueryRowContext(ctx, createAPIToken, userID, label, tokenHash)
	var t APIToken
	err := row.Scan(&t.ID, &t.UserID, &t.Label, &t.TokenHash, &t.CreatedAt, &t.LastUsedAt)
	return t, err
}

const listAPITokens = `
select id, user_id, label, token_hash, created_at, last_used_at
from api_tokens
order by created_at desc
`

func (q *Queries) ListAPITokens(ctx context.Context) ([]APIToken, error) {
	rows, err := q.db.QueryContext(ctx, listAPITokens)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []APIToken
	for rows.Next() {
		var t APIToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Label, &t.TokenHash, &t.CreatedAt, &t.LastUsedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

const getAPITokenByHash = `
select id, user_id, label, token_hash, created_at, last_used_at
from api_tokens
where token_hash = $1
`

func (q *Queries) GetAPITokenByHash(ctx context.Context, tokenHash string) (APIToken, error) {
	row := q.db.QueryRowContext(ctx, getAPITokenByHash, tokenHash)
	var t APIToken
	err := row.Scan(&t.ID, &t.UserID, &t.Label, &t.TokenHash, &t.CreatedAt, &t.LastUsedAt)
	if err == sql.ErrNoRows {
		return t, sql.ErrNoRows
	}
	return t, err
}

const touchAPITokenLastUsed = `
update api_tokens set last_used_at = now() where id = $1
`

func (q *Queries) TouchAPITokenLastUsed(ctx context.Context, id string) error {
	_, err := q.db.ExecContext(ctx, touchAPITokenLastUsed, id)
	return err
}

const deleteAPIToken = `
delete from api_tokens where id = $1
`

func (q *Queries) DeleteAPIToken(ctx context.Context, id string) error {
	_, err := q.db.ExecContext(ctx, deleteAPIToken, id)
	return err
}
