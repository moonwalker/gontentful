package schema

const Gamesbrowser = `
{{ if $.SchemaName }}
CREATE SCHEMA IF NOT EXISTS {{ $.SchemaName }};
{{ end }}
CREATE TABLE IF NOT EXISTS _game_meta (
	slug text primary key,
	provider text,
	studio text,
	category text,
	format text,
	type text,
	bonus_features text[] not null default '{}',
	labels text[] not null default '{}',
	tags text[] not null default '{}',
	themes text[] not null default '{}',
	win_features text[] not null default '{}',
	wild_features text[] not null default '{}',
	payout_properties jsonb not null default '{}'::jsonb,
	screens text[] not null default '{}',
	settings text[] not null default '{}', 
	provider_ids jsonb not null default '{}'::jsonb,	
	display_ratios jsonb not null default '{}'::jsonb,
	priority integer not null default 0,
	excluded_markets text[] not null default '{}',
	content text not null unique,
	game_identifier text,
	sys_id text not null unique,
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone,
	deleted_by text
);

CREATE TABLE IF NOT EXISTS _game_content (
	sys_id text primary key,
	content jsonb not null default '{}',
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone,
	deleted_by text
);

ALTER TABLE _game_meta DROP CONSTRAINT IF EXISTS gamesbrowser_content_fkey;

ALTER TABLE _game_meta
  ADD CONSTRAINT gamesbrowser_content_fkey
  FOREIGN KEY (content)
  REFERENCES _game_content (sys_id)
  ON DELETE CASCADE;

CREATE UNIQUE INDEX IF NOT EXISTS game_sys_id ON _game_meta (sys_id);`
