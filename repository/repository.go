package repository

import (
	"context"
	"errors"
	"fmt"

	"url/link"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("link not found")
var ErrCodeTaken = errors.New("code already taken")
var ErrAliasTaken = errors.New("alias already taken")

var _ ManagerRepository = (*LinkRepository)(nil)

type ManagerRepository interface {
	Create(ctx context.Context, link *link.Link) error
	GetByCode(ctx context.Context, code string) (*link.Link, error)
	DeleteLinkByCode(ctx context.Context, code string) error
	GetAllLinks(ctx context.Context) ([]link.Link, error)
	GetAndIncrement(ctx context.Context, code string) (*link.Link, error)
}

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
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "links_code_key" {
				return ErrCodeTaken
			}
		}

		return fmt.Errorf("Create link error: %w", err)
	}

	return nil
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
		return nil, fmt.Errorf("Get link error: %w", err)
	}

	return &newLink, nil
}

func (r *LinkRepository) DeleteLinkByCode(ctx context.Context, code string) error {

	query := `
	    DELETE FROM links
		WHERE code = $1
	`

	result, err := r.db.Exec(
		ctx,
		query,
		code,
	)
	if err != nil {
		return fmt.Errorf("delete link: %w", err)
	}

	count := result.RowsAffected()
	if count == 0 {
		return ErrNotFound
	}

	return nil
}

func (r LinkRepository) UpdateLink(ctx context.Context, code string) error {

	query := `
	    UPDATE links
		SET clicks = clicks + 1
		WHERE code = $1;
	`

	result, err := r.db.Exec(
		ctx,
		query,
		code,
	)
	if err != nil {
		return fmt.Errorf("update link %s: %w", code, err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("update link %s: %w", code, ErrNotFound)
	}

	return nil
}

func (r *LinkRepository) GetAllLinks(ctx context.Context) ([]link.Link, error) {

	query := `
	SELECT 
	    code,
		clicks,
		original_url,
		COALESCE(alias, '') AS alias,
		created_at,
		expires_at
	FROM links
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var links []link.Link

	for rows.Next() {

		var link link.Link

		err := rows.Scan(
			&link.Code,
			&link.Clicks,
			&link.OriginalURL,
			&link.Alias,
			&link.CreatedAt,
			&link.ExpiresAt,
		)
		if err != nil {
			return nil, err
		}

		links = append(links, link)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}

func (r *LinkRepository) GetAndIncrement(ctx context.Context, code string) (*link.Link, error) {

	query := `
	UPDATE links
	SET clicks = clicks + 1
	WHERE code = $1
	   AND (
	       expires_at IS NULL
		   OR expires_at > NOW()  
	   )
	RETURNING code, original_url, expires_at;
	`

	var newLink link.Link

	err := r.db.QueryRow(
		ctx,
		query,
		code,
	).Scan(
		&newLink.Code,
		&newLink.OriginalURL,
		&newLink.ExpiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("error %s: %w", code, err)
	}

	return &newLink, nil
}
