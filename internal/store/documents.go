package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrNotFound is returned by single-row lookups with no match.
var ErrNotFound = errors.New("not found")

// ArticleSummary is a published article as listings need it: no body, tags
// pre-aggregated in name order.
type ArticleSummary struct {
	ID          int64     `db:"id"`
	Slug        string    `db:"slug"`
	Title       string    `db:"title"`
	Author      string    `db:"author"`
	PublishedAt time.Time `db:"published_at"`
	TagSlugs    []string  `db:"tag_slugs"`
	TagNames    []string  `db:"tag_names"`
}

// Article is a full article row for the article page.
type Article struct {
	ID          int64     `db:"id"`
	Slug        string    `db:"slug"`
	Title       string    `db:"title"`
	Author      string    `db:"author"`
	BodyHTML    string    `db:"body_html"`
	PublishedAt time.Time `db:"published_at"`
	TagSlugs    []string  `db:"tag_slugs"`
	TagNames    []string  `db:"tag_names"`
}

// Category is a row of the categories table.
type Category struct {
	ID   int64  `db:"id"`
	Slug string `db:"slug"`
	Name string `db:"name"`
}

// Doc is a docs page joined with its project.
type Doc struct {
	ID          int64   `db:"id"`
	Slug        string  `db:"slug"`
	Title       string  `db:"title"`
	BodyHTML    string  `db:"body_html"`
	Version     *string `db:"version"`
	ProjectSlug string  `db:"project_slug"`
	ProjectName string  `db:"project_name"`
}

const articleListSQL = `
	select d.id, d.slug, d.title, d.author, d.published_at,
	       coalesce(array_agg(c.slug order by c.name) filter (where c.id is not null), '{}') as tag_slugs,
	       coalesce(array_agg(c.name order by c.name) filter (where c.id is not null), '{}') as tag_names
	from documents d
	left join document_categories dc on dc.document_id = d.id
	left join categories c on c.id = dc.category_id
	where d.kind = 'article' and d.latest and d.published_at is not null`

// ListPublishedArticles returns every published article, newest first.
func (s *Store) ListPublishedArticles(ctx context.Context) ([]ArticleSummary, error) {
	rows, err := s.pool.Query(ctx, articleListSQL+`
		group by d.id
		order by d.published_at desc`)
	if err != nil {
		return nil, fmt.Errorf("list articles: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[ArticleSummary])
}

// ListArticlesByTag returns published articles carrying the tag, newest
// first.
func (s *Store) ListArticlesByTag(ctx context.Context, tagSlug string) ([]ArticleSummary, error) {
	rows, err := s.pool.Query(ctx, articleListSQL+`
		and exists (
			select 1
			from document_categories dc2
			join categories c2 on c2.id = dc2.category_id
			where dc2.document_id = d.id and c2.slug = $1
		)
		group by d.id
		order by d.published_at desc`, tagSlug)
	if err != nil {
		return nil, fmt.Errorf("list articles by tag: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[ArticleSummary])
}

// GetArticle returns one published article by slug.
func (s *Store) GetArticle(ctx context.Context, slug string) (Article, error) {
	rows, err := s.pool.Query(ctx, `
		select d.id, d.slug, d.title, d.author, d.body_html, d.published_at,
		       coalesce(array_agg(c.slug order by c.name) filter (where c.id is not null), '{}') as tag_slugs,
		       coalesce(array_agg(c.name order by c.name) filter (where c.id is not null), '{}') as tag_names
		from documents d
		left join document_categories dc on dc.document_id = d.id
		left join categories c on c.id = dc.category_id
		where d.kind = 'article' and d.latest and d.published_at is not null and d.slug = $1
		group by d.id`, slug)
	if err != nil {
		return Article{}, fmt.Errorf("get article: %w", err)
	}
	article, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Article])
	if errors.Is(err, pgx.ErrNoRows) {
		return Article{}, ErrNotFound
	}
	return article, err
}

// GetCategory returns one category by slug.
func (s *Store) GetCategory(ctx context.Context, slug string) (Category, error) {
	rows, err := s.pool.Query(ctx, `
		select id, slug, name from categories where slug = $1`, slug)
	if err != nil {
		return Category{}, fmt.Errorf("get category: %w", err)
	}
	category, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Category])
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	return category, err
}

// GetDoc returns one docs page by project and doc slug, latest version.
func (s *Store) GetDoc(ctx context.Context, projectSlug, docSlug string) (Doc, error) {
	rows, err := s.pool.Query(ctx, `
		select d.id, d.slug, d.title, d.body_html, d.version,
		       p.slug as project_slug, p.name as project_name
		from documents d
		join projects p on p.id = d.project_id
		where d.kind = 'doc' and d.latest and p.slug = $1 and d.slug = $2`,
		projectSlug, docSlug)
	if err != nil {
		return Doc{}, fmt.Errorf("get doc: %w", err)
	}
	doc, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Doc])
	if errors.Is(err, pgx.ErrNoRows) {
		return Doc{}, ErrNotFound
	}
	return doc, err
}

// UpsertDoc writes one synced docs page, keyed by project slug and doc slug.
func (s *Store) UpsertDoc(ctx context.Context, projectSlug, slug, title, bodyMD, bodyHTML string, renderVersion int) error {
	tag, err := s.pool.Exec(ctx, `
		insert into documents (kind, project_id, slug, title, author, body_md, body_html, render_version, published_at)
		select 'doc', p.id, $2, $3, 'sync', $4, $5, $6, now()
		from projects p
		where p.slug = $1
		on conflict (kind, project_id, slug, version) do update
		set title          = excluded.title,
		    body_md        = excluded.body_md,
		    body_html      = excluded.body_html,
		    render_version = excluded.render_version,
		    updated_at     = now()`,
		projectSlug, slug, title, bodyMD, bodyHTML, renderVersion)
	if err != nil {
		return fmt.Errorf("upsert doc %s/%s: %w", projectSlug, slug, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("upsert doc %s/%s: unknown project", projectSlug, slug)
	}
	return nil
}

// PruneDocs deletes docs rows absent from keep, whose entries are
// "projectSlug/docSlug". Rows for renamed or removed pages go away with the
// sync that dropped them.
func (s *Store) PruneDocs(ctx context.Context, keep []string) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		delete from documents d
		using projects p
		where d.project_id = p.id
		  and d.kind = 'doc'
		  and not (p.slug || '/' || d.slug = any($1))`, keep)
	if err != nil {
		return 0, fmt.Errorf("prune docs: %w", err)
	}
	return tag.RowsAffected(), nil
}

// ListProjects returns every project ordered by name.
func (s *Store) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := s.pool.Query(ctx, `
		select id, family_id, slug, name, description, version, repo_url
		from projects
		order by name`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[Project])
}
