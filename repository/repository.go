package repository

import (
	"context"
	"fmt"
	"log"

	"url/link"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LinkRepository struct {
	db *pgxpool.Pool
}

func NewPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("Создайте пулл соединений: %w", err)
	}

	return pool, nil
}

func NewLinkRepository(db *pgxpool.Pool) *LinkRepository {
	return &LinkRepository{
		db: db,
	}
}

func (r *LinkRepository) Create(ctx context.Context, link *link.Link) error {

	query := `
	    INSERT INTO links (
		   code,
		   original_url,
		   expires_at
		)
		VALUES ($1, $2, $3)
		RETURNING created_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		link.Code,
		link.OriginalURL,
		link.ExpiresAt,
	).Scan(
		&link.CreatedAt,
	)

	return err
}

func (r *LinkRepository) GetByCode(ctx context.Context, code string) (*link.Link, error) {
	query := `
	    SELECT
		     code,
		     original_url,
		     clicks,
		     created_at,
		     expires_at
		FROM links
		WHERE code = $1
	`

	var newLink link.Link

	err := r.db.QueryRow(ctx, query, code).Scan(
		&newLink.Code,
		&newLink.OriginalURL,
		&newLink.Clicks,
		&newLink.CreatedAt,
		&newLink.ExpiresAt,
	)

	if err != nil {
		return nil, err
	}

	return &newLink, nil
}

func (r *LinkRepository) DeleteLinkByCode(ctx context.Context, code string) error {

	query := `
	    DELETE FROM links
		WHERE code = $1
	`

	_, err := r.db.Exec(
		ctx,
		query,
		code,
	)
	if err != nil {
		log.Println("Не удалось удалить строку:", err)
		return err
	}

	return nil
}

func (r LinkRepository) UpdateLink(ctx context.Context, code string) error {

	query := `
	    UPDATE 
		clicks = clicks + 1
		FROM links
		WHERE code = $1
	`

	_, err := r.db.Exec(
		ctx,
		query,
		code,
	)
	if err != nil {
		return err
	}

	return nil
}
