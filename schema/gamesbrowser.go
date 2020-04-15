package schema

const Gamesbrowser = `
{{ if $.SchemaName }}
CREATE SCHEMA IF NOT EXISTS {{ $.SchemaName }};
--
{{- end -}}
CREATE TABLE IF NOT EXISTS gamesbrowser_meta (
	slug text primary key,
	provider text not null,
	studio text not null,
	category text,
	format text,
	type text,
	bonus_features text[] not null default '{}',
	labels text[] not null default '{}',
	tags text[] not null default '{}',
	themes text[] not null default '{}',
	win_features text[] not null default '{}',
	wild_features text[] not null default '{}',
	payout_properties text[] not null default '{}'
	screens text[] not null default '{}',
	settings text[] not null default '{}', 
	provider_ids jsonb not null default '{}'::jsonb,
	priority integer not null default 0,
	excluded_markets text[] not null default '{}',
	content text not null unique,
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone default now(),
	deleted_by text not null default 'system'
);

CREATE TABLE IF NOT EXISTS gamesbrowser_content (
	slug text primary key,
	content jsonb not null default '{}',
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone default now(),
	deleted_by text not null default 'system'
);

ALTER TABLE gamesbrowser_meta DROP CONSTRAINT IF EXISTS gamesbrowser_content_fkey;

ALTER TABLE gamesbrowser_meta
  ADD CONSTRAINT gamesbrowser_content_fkey
  FOREIGN KEY content
  REFERENCES gamesbrowser_content (slug)
  ON DELETE CASCADE;
`
