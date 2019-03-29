package gontentful

const createSchemaTemplate = `BEGIN;
CREATE SCHEMA IF NOT EXISTS {{ $.SchemaName }};
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $.AssetTableName }} (
	id text primary key not null,
	fields jsonb,
	type text not null,
	revision integer not null default 0,
	version integer not null default 0,
	published_version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null,
	published_at timestamp without time zone,
	published_by text
  );
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $.ModelsTableName }} (
	id text primary key not null,
	name text not null,
	description text,
	display_field text not null,
	revision integer not null default 0,
	version integer not null default 0,
	published_version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null,
	published_at timestamp without time zone,
	published_by text
);
--
COMMIT;
`

const createMetaTableTemplate = `
BEGIN
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ .TableName }}_meta (
	name text primary key not null,
	type text not null,
	link_type text,
	items jsonb,
	is_localized boolean default false,
	is_required boolean default false,
	is_disabled boolean default false,
	is_omitted boolean default false,
	validations jsonb not null default '[]',
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null
);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ .TableName }}_meta_history (
	name text primary key not null,
	type text not null,
	link_type text,
	items jsonb,
	is_localized boolean default false,
	is_required boolean default false,
	is_disabled boolean default false,
	is_omitted boolean default false,
	validations jsonb not null default '[]',
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null
	archived_at timestamp without time zone,
	archived_by text
);
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}.on_{{ .TableName }}_meta_update()
RETURNS void AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}_meta_history (
		name,
		type,
		link_type,
		items,
		is_localized,
		is_required,
		is_disabled,
		is_omitted,
		validations,
		created_at,
		created_by,
		updated_at,
		updated_by,
		archived_by
	) VALUES (
		OLD.name,
		OLD.type,
		OLD.link_type,
		OLD.items,
		OLD.is_localized,
		OLD.is_required,
		OLD.is_disabled,
		OLD.is_omitted,
		OLD.validations,
		OLD.created_at,
		OLD.created_by,
		OLD.updated_at,
		OLD.updated_by,
		NEW.updated_by
	);
	COMMIT;
END;
$$ LANGUAGE plpgsql;
--
CREATE OR REPLACE TRIGGER {{ $.SchemaName }}.{{ .TableName }}_meta_update
    AFTER UPDATE ON {{ $.SchemaName }}.{{ .TableName }}
    FOR EACH ROW
	EXECUTE FUNCTION {{ $.SchemaName }}.on_{{ .TableName }}_meta_update();
--
COMMIT;
`

const createContentTableTemplate = `
BEGIN
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ .TableName }} (
	id text primary key not null,
	fields jsonb not null default '[]'::jsonb,
	type text not null,
	revision integer not null default 0,
	version integer not null default 0,
	published_version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null,
	published_at timestamp without time zone,
	published_by text
);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ .TableName }}_history (
	hid serial primary key,
	id text not null,
	fields jsonb not null default '[]'::jsonb,
	type text not null,
	revision integer not null default 0,
	version integer not null default 0,
	published_version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null,
	published_at timestamp without time zone,
	published_by text
	archived_at timestamp without time zone,
	archived_by text
);
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}.on_{{ .TableName }}_update()
RETURNS void AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}_history (
		id,
		fields,
		type,
		revision,
		version,
		published_version,
		created_at,
		created_by,
		updated_at,
		updated_by,
		published_at,
		published_by,
		archived_by
	) VALUES (
		OLD.id,
		OLD.fields,
		OLD.type,
		OLD.revision,
		OLD.version,
		OLD.published_version,
		OLD.created_at,
		OLD.created_by,
		OLD.updated_at,
		OLD.updated_by,
		OLD.published_at,
		OLD.published_by,
		NEW.updated_by
	);
	COMMIT;
END;
$$ LANGUAGE plpgsql;
--
CREATE OR REPLACE TRIGGER {{ $.SchemaName }}.{{ .TableName }}_update
    AFTER UPDATE ON {{ $.SchemaName }}.{{ .TableName }}
    FOR EACH ROW
	EXECUTE FUNCTION {{ $.SchemaName }}.on_{{ .TableName }}_update();
--
COMMIT;
`

const contentInsertRowTemplate = `
{{ range $tblidx, $tbl := $.Tables }}
BEGIN;
{{ range $itemidx, $item := .Rows }}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }} (
	id,
	fields,
	type,
	revision,
	version,
	published_version,
	created_at,
	created_by,
	updated_at,
	updated_by,
	published_at,
	published_by
) VALUES (
	'{{ .ID }}',
	{{ if .Fields }}'{{ .Fields }}'::jsonb{{ else }}NULL{{ end }},
	'{{ .Type }}',
	{{ .Revision }},
	{{ .Version }},
	{{ .PublishedVersion }},
	to_timestamp('{{ .CreatedAt }}', 'YYYY-MM-DDThh24:mi:ss.mssZ'),
	'system',
	to_timestamp('{{ .UpdatedAt }}', 'YYYY-MM-DDThh24:mi:ss.mssZ'),
	'system',
	{{ if .PublishedAt }}to_timestamp('{{ .PublishedAt }}','YYYY-MM-DDThh24:mi:ss.mssZ'){{ else }}NULL{{ end }},
	'{{ .PublishedBy }}'
)
ON CONFLICT (id) DO UPDATE
SET
	fields = EXCLUDED.fields,
	type = EXCLUDED.type,
	revision = EXCLUDED.revision,
	version = EXCLUDED.version,
	published_version = EXCLUDED.published_version,
	updated_at = EXCLUDED.updated_at,
	updated_by = EXCLUDED.updated_by,
	published_at = EXCLUDED.published_at,
	published_by = EXCLUDED.published_by
RETURNING 1;
--
{{ end -}}
COMMIT;
{{ end -}}`
