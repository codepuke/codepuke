package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// AdminArticleRow is one line of the admin article list: every article,
// drafts included.
type AdminArticleRow struct {
	ID          int64      `db:"id"`
	Slug        string     `db:"slug"`
	Title       string     `db:"title"`
	Author      string     `db:"author"`
	PublishedAt *time.Time `db:"published_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// AdminArticle is the full editable article for the editor.
type AdminArticle struct {
	ID          int64      `db:"id"`
	Slug        string     `db:"slug"`
	Title       string     `db:"title"`
	Author      string     `db:"author"`
	BodyMD      string     `db:"body_md"`
	BodyHTML    string     `db:"body_html"`
	PublishedAt *time.Time `db:"published_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	TagSlugs    []string   `db:"tag_slugs"`
}

// ListArticlesAdmin returns every article, most recently touched first.
func (s *Store) ListArticlesAdmin(ctx context.Context) ([]AdminArticleRow, error) {
	rows, err := s.pool.Query(ctx, `
		select id, slug, title, author, published_at, updated_at
		from documents
		where kind = 'article'
		order by updated_at desc`)
	if err != nil {
		return nil, fmt.Errorf("list articles admin: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[AdminArticleRow])
}

// GetArticleByID returns one article, draft or published, for editing.
func (s *Store) GetArticleByID(ctx context.Context, id int64) (AdminArticle, error) {
	rows, err := s.pool.Query(ctx, `
		select d.id, d.slug, d.title, d.author, d.body_md, d.body_html, d.published_at, d.updated_at,
		       coalesce(array_agg(c.slug order by c.slug) filter (where c.id is not null), '{}') as tag_slugs
		from documents d
		left join document_categories dc on dc.document_id = d.id
		left join categories c on c.id = dc.category_id
		where d.kind = 'article' and d.id = $1
		group by d.id`, id)
	if err != nil {
		return AdminArticle{}, fmt.Errorf("get article by id: %w", err)
	}
	article, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[AdminArticle])
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminArticle{}, ErrNotFound
	}
	return article, err
}

// CreateArticle inserts an empty draft and returns its id. The slug is fixed
// at creation so published links never move under edits.
func (s *Store) CreateArticle(ctx context.Context, slug, title, author string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		insert into documents (kind, slug, title, author, body_md, body_html, render_version)
		values ('article', $1, $2, $3, '', '', $4)
		returning id`, slug, title, author, 0).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create article: %w", err)
	}
	return id, nil
}

// UpdateArticle saves the editable fields plus the fresh render, and resets
// the article's categories to tags (slugs; unknown ones are created with a
// de-hyphenated display name).
func (s *Store) UpdateArticle(ctx context.Context, id int64, title, author, bodyMD, bodyHTML string, renderVersion int, tags []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("update article: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		update documents
		set title = $2, author = $3, body_md = $4, body_html = $5,
		    render_version = $6, updated_at = now()
		where kind = 'article' and id = $1`,
		id, title, author, bodyMD, bodyHTML, renderVersion)
	if err != nil {
		return fmt.Errorf("update article: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if _, err := tx.Exec(ctx, `delete from document_categories where document_id = $1`, id); err != nil {
		return fmt.Errorf("update article tags: %w", err)
	}
	for _, slug := range tags {
		if _, err := tx.Exec(ctx, `
			insert into categories (slug, name)
			values ($1, $2)
			on conflict (slug) do nothing`, slug, strings.ReplaceAll(slug, "-", " ")); err != nil {
			return fmt.Errorf("upsert category %s: %w", slug, err)
		}
		if _, err := tx.Exec(ctx, `
			insert into document_categories (document_id, category_id)
			select $1, id from categories where slug = $2`, id, slug); err != nil {
			return fmt.Errorf("tag article %s: %w", slug, err)
		}
	}

	return tx.Commit(ctx)
}

// SetPublished publishes (at non-nil) or unpublishes (nil) an article.
func (s *Store) SetPublished(ctx context.Context, id int64, at *time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		update documents
		set published_at = $2, updated_at = now()
		where kind = 'article' and id = $1`, id, at)
	if err != nil {
		return fmt.Errorf("set published: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
