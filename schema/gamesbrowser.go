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
	release_date text,
	fee_group text,
	provider_ids jsonb not null default '{}'::jsonb,	
	priority integer not null default 0,
	excluded_markets text[] not null default '{}',
	enabled boolean default FALSE,
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone,
	deleted_by text
);

CREATE TABLE IF NOT EXISTS _game_content (
	slug text primary key,
	content jsonb not null default '{}',
	sys_id text not null unique,
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone,
	deleted_by text
);

CREATE TABLE IF NOT EXISTS _game_history (
	version text primary key,
	timestamp text not null,
	slug text not null,
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
	release_date text,
	fee_group text,
	provider_ids jsonb not null default '{}'::jsonb,	
	priority integer not null default 0,
	excluded_markets text[] not null default '{}',
	enabled boolean default FALSE,
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone,
	deleted_by text
);

CREATE TABLE IF NOT EXISTS _csv_upload (
	timestamp text,
	csv text not null,
	count integer not null default 0,
	type text,
	success bool not null default FALSE,
	reason text,
	created timestamp without time zone default now(),
	created_by text not null default 'system'
);

ALTER TABLE _csv_upload ADD COLUMN IF NOT EXISTS reason text NULL;


CREATE TABLE IF NOT EXISTS _studio_excluded_market (
	studio text primary key,
	excluded_markets text[] not null default '{}',
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone,
	deleted_by text
);

CREATE TABLE IF NOT EXISTS _studio_excluded_currency (
	studio text primary key,
	excluded_currencies text[] not null default '{}',
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone,
	deleted_by text
);

CREATE TABLE IF NOT EXISTS _studio_excluded_licence (
	studio text primary key,
	excluded_licences text[] not null default '{}',
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone,
	deleted_by text
);

CREATE TABLE IF NOT EXISTS _provider_studio_required_fields (
	studio text primary key,
	required_fields text[] not null default '{}',
	created timestamp without time zone default now(),
	created_by text not null default 'system',
	updated timestamp without time zone default now(),
	updated_by text not null default 'system',
	deleted timestamp without time zone,
	deleted_by text
);

CREATE UNIQUE INDEX IF NOT EXISTS game_sys_id ON _game_content (sys_id);
`

// ALTER TABLE _game_meta DROP CONSTRAINT IF EXISTS gamesbrowser_content_fkey;

// ALTER TABLE _game_meta
//   ADD CONSTRAINT gamesbrowser_content_fkey
//   FOREIGN KEY (slug)
//   REFERENCES _game_content (slug)
//   ON DELETE CASCADE;
