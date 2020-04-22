package schema

const Gamesbrowser = `
{{ if $.SchemaName }}
CREATE SCHEMA IF NOT EXISTS {{ $.SchemaName }};
{{ end }}
CREATE TABLE IF NOT EXISTS gamesbrowser_meta (
	slug text primary key,
	provider text,
	studio text,
	category text,
	format text,
	type text,
	bonus_features text[] default '{}',
	labels text[] default '{}',
	tags text[] default '{}',
	themes text[] default '{}',
	win_features text[] default '{}',
	wild_features text[] default '{}',
	payout_properties text,
	screens text[] default '{}',
	settings text[] default '{}', 
	provider_ids jsonb default '{}'::jsonb,
	priority integer default 0,
	excluded_markets text[] default '{}',
	content text not null unique,
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone,
	deleted_by text
);

CREATE TABLE IF NOT EXISTS gamesbrowser_content (
	slug text primary key,
	content jsonb not null default '{}',
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone,
	deleted_by text
);

ALTER TABLE gamesbrowser_meta DROP CONSTRAINT IF EXISTS gamesbrowser_content_fkey;

ALTER TABLE gamesbrowser_meta
  ADD CONSTRAINT gamesbrowser_content_fkey
  FOREIGN KEY (content)
  REFERENCES gamesbrowser_content (slug)
  ON DELETE CASCADE;
`
