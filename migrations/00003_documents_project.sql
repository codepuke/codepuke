-- +goose Up
-- Docs belong to a project; articles do not. Two projects can both ship an
-- "overview" doc, so project_id joins the identity constraint. Articles keep
-- a null project_id, and nulls-not-distinct still forbids two articles with
-- the same slug.
alter table documents
    add column project_id bigint references projects (id);

alter table documents
    drop constraint documents_kind_slug_version_key;

alter table documents
    add constraint documents_identity_key
    unique nulls not distinct (kind, project_id, slug, version);

-- +goose Down
alter table documents
    drop constraint documents_identity_key;

alter table documents
    add constraint documents_kind_slug_version_key
    unique nulls not distinct (kind, slug, version);

alter table documents
    drop column project_id;
