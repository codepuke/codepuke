-- +goose Up
create table families (
    id         bigint generated always as identity primary key,
    name       text not null unique,
    descriptor text not null
);

create table projects (
    id          bigint generated always as identity primary key,
    family_id   bigint not null references families (id),
    slug        text not null unique,
    name        text not null,
    description text not null,
    version     text,
    repo_url    text
);

create table documents (
    id             bigint generated always as identity primary key,
    kind           text not null check (kind in ('article', 'doc')),
    slug           text not null,
    title          text not null,
    author         text not null,
    body_md        text not null,
    body_html      text not null,
    render_version int not null,
    version        text,
    latest         boolean not null default true,
    published_at   timestamptz,
    updated_at     timestamptz not null default now(),
    unique nulls not distinct (kind, slug, version)
);

create index documents_published_idx
    on documents (published_at desc)
    where published_at is not null;

create table categories (
    id   bigint generated always as identity primary key,
    slug text not null unique,
    name text not null
);

create table document_categories (
    document_id bigint not null references documents (id) on delete cascade,
    category_id bigint not null references categories (id) on delete cascade,
    primary key (document_id, category_id)
);

-- +goose Down
drop table document_categories;
drop table categories;
drop table documents;
drop table projects;
drop table families;
