package database


import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LinkRepository struct {
	db *pgxpool.Pool
}

CREATE TABLE links (
	id BIGSERIAL PRIMARY KEY,
	code TEXT UNIQUE NOT NULL,
	original_url TEXT UNIQUE NOT NULL,
	clicks BIGINT NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DAFULT NOW(),
	expires_at TIMESTAMPTZ
);


func NewPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		retunr nil, fmt.Errorf("Создайте пулл соединений: %w", err)
	}

	return pool, nil
}

func NewLinkRepository(db *pgxpool.Pool) *Linkrepository {
	return &LinkRepository{
		db : db,
	}
}

func (r *Linkrepository) Create(ctx context.Context, link *Link,) error {

	query := `
	    INSERT INTO links (
		   code,
		   original_url,
		   expires_at
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		link.Code,
		link.OriginalURL,
		link.ExpiresAt,
	).Scan(
		&link.ID,
		&link.CreatedAt,
	)

	return err
}

func (r *LinkRepository) GetByCode(ctx *pgxpool.Pool, code string) (*Link, error) {
	query := `
	    SELECT
		     id,
		     code,
		     original_url,
		     clicks,
		     created_at,
		     expires_at
		FROM links
		WHERE code = $1
	`

	var link *Link

	err := r.db.QueryRow(ctx, query, code,).Scan(
		&link.ID,
		&link.Code,
		&link.OriginalURL,
		&link.Clicks,
		&link.CreatedAt,
		&link.expiresAt,
	)

	if err != nil {
		retunr nil, err
	}

	return &link, nil
}
