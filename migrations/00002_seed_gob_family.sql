-- +goose Up
insert into families (name, descriptor)
values ('gob', 'the encoding/gob wire format, inspected and ported');

insert into projects (family_id, slug, name, description, repo_url)
select f.id, p.slug, p.name, p.description, p.repo_url
from families f
join (
    values
        ('gobspect', 'gobspect',
         'decode-only introspection for gob streams, no original types required',
         'https://github.com/codepuke/gobspect'),
        ('gq', 'gq',
         'jq-inspired cli for inspecting gob streams from the terminal',
         'https://github.com/codepuke/gobspect/tree/main/cmd/gq'),
        ('gobspect-mcp', 'gobspect-mcp',
         'mcp server exposing gob stream inspection as structured tool calls',
         'https://github.com/codepuke/gobspect-mcp'),
        ('gobts', 'gobts',
         'typescript port of encoding/gob, wire-compatible both directions',
         'https://github.com/codepuke/gobts'),
        ('pygob', 'pygob',
         'python port of encoding/gob, wire-compatible both directions',
         'https://github.com/codepuke/pygob'),
        ('gobdotnet', 'gobdotnet',
         'pure c# encoder and decoder for the gob wire format',
         'https://github.com/codepuke/gobdotnet')
) as p (slug, name, description, repo_url) on true
where f.name = 'gob';

-- +goose Down
delete from projects
where family_id = (select id from families where name = 'gob');

delete from families
where name = 'gob';
