package gontentful

const pgTemplate = `BEGIN;
CREATE SCHEMA IF NOT EXISTS {{ .SchemaName }};
--
CREATE TABLE IF NOT EXISTS {{ .SchemaName }}._space (
	_id serial primary key,
	spaceId text not null unique,
	name text not null,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null
);
CREATE UNIQUE INDEX IF NOT EXISTS spaceId ON {{ .SchemaName }}._space(spaceId);
--
CREATE TABLE IF NOT EXISTS {{ .SchemaName }}._locales (
	_id serial primary key,
	code text not null unique,
	name text not null,
	isDefault boolean,
	fallbackCode text,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null
);
CREATE UNIQUE INDEX IF NOT EXISTS code ON {{ .SchemaName }}._locales(code);
--
{{ range $locidx, $loc := $.Space.Locales }}
INSERT INTO {{ $.SchemaName }}._locales (
	code,
	name,
	isDefault,
	fallbackCode,
	created_by,
	updated_by
) VALUES (
	'{{ .Code }}',
	'{{ .Name }}',
	{{ .Default }},
	'{{ .FallbackCode }}',
	'system',
	'system'
)
ON CONFLICT (code) DO UPDATE
SET
	name = EXCLUDED.name,
	isDefault = EXCLUDED.isDefault,
	fallbackCode = EXCLUDED.fallbackCode,
	updated_at = EXCLUDED.updated_at,
	updated_by = EXCLUDED.updated_by
;
{{ end }}
--
CREATE TABLE IF NOT EXISTS {{ .SchemaName }}._models (
	_id serial primary key,
	name text not null unique,
	label text not null,
	description text,
	display_field text not null,
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS name ON {{ $.SchemaName }}._models(name);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._models__history(
	_id serial primary key,
	pubId integer not null,
	name text not null,
	fields jsonb not null,
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null
);
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}.on__models_update()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}._models__history (
		pubId,
		name,
		fields,
		version,
		created_by
	) VALUES (
		OLD._id,
		OLD.name,
		row_to_json(OLD),
		OLD.version,
		NEW.updated_by
	);
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}__models_update ON {{ $.SchemaName }}._models;
--
CREATE TRIGGER {{ $.SchemaName }}__models_update
    AFTER UPDATE ON {{ $.SchemaName }}._models
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on__models_update();
--
CREATE TABLE IF NOT EXISTS {{ .SchemaName }}._assets (
	_id serial primary key,
	sysId text not null unique,
	title text not null,
	description text,
	fileName text,
	contentType text,
	url text,
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS sysId ON {{ .SchemaName }}._assets(sysId);
--
CREATE OR REPLACE FUNCTION assets_upsert(_sysId text, _title text, _description text, _fileName text, _contentType text, _url text, _version integer, _created_at timestamp, _created_by text, _updated_at timestamp, _updated_by text)
RETURNS void AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}._assets (
	sysId,
	title,
	description,
	fileName,
	contentType,
	url,
	version,
	created_at,
	created_by,
	updated_at,
	updated_by
) VALUES (
	_sysId,
	_title,
	_description,
	_fileName,
	_contentType,
	_url,
	_version,
	_createdAt,
	_createdBy,
	_updatedAt,
	_updatedBy
)
ON CONFLICT (sysId) DO UPDATE
SET
	title = EXCLUDED.title,
	description = EXCLUDED.description,
	fileName = EXCLUDED.fileName,
	contentType = EXCLUDED.contentType,
	url = EXCLUDED.url,
	version = EXCLUDED.version,
	updated_at = now(),
	updated_by = EXCLUDED.updated_by
;
END;
$$  LANGUAGE plpgsql;
--
CREATE TABLE IF NOT EXISTS {{ .SchemaName }}._assets__publish (
	_id serial primary key,
	sysId text not null unique,
	title text not null,
	description text,
	fileName text,
	contentType text,
	url text,
	version integer not null default 0,
	published_at timestamp without time zone default now(),
	published_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS sysId ON {{ .SchemaName }}._assets__publish(sysId);
--
CREATE OR REPLACE FUNCTION assets_publish(_aid integer)
RETURNS void AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}._assets__publish (
	sysId,
	title,
	description,
	fileName,
	contentType,
	url,
	version,
	published_by
)
SELECT
	sysId,
	title,
	description,
	fileName,
	contentType,
	url,
	version,
	updated_by
FROM {{ $.SchemaName }}._assets
WHERE _id = _aid
ON CONFLICT (sysId) DO UPDATE
SET
	title = EXCLUDED.title,
	description = EXCLUDED.description,
	fileName = EXCLUDED.fileName,
	contentType = EXCLUDED.contentType,
	url = EXCLUDED.url,
	version = EXCLUDED.version,
	published_at = now(),
	published_by = EXCLUDED.published_by
;
END;
$$  LANGUAGE plpgsql;
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._assets__history(
	_id serial primary key,
	pubId integer not null,
	sysId text not null,
	fields jsonb not null,
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null
);
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}.on__assets_update()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}._assets__history (
		pubId,
		sysId,
		fields,
		version,
		created_by
	) VALUES (
		OLD._id,
		OLD.sysId,
		row_to_json(OLD),
		OLD.version,
		NEW.updated_by
	);
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}__assets_update ON {{ $.SchemaName }}._assets__publish;
--
CREATE TRIGGER {{ $.SchemaName }}__assets_update
    AFTER UPDATE ON {{ $.SchemaName }}._assets__publish
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on__assets_update();
COMMIT;
----
{{ range $tblidx, $tbl := $.Tables }}
BEGIN;
INSERT INTO {{ $.SchemaName }}._models (
	name,
	label,
	description,
	display_field,
	version,
	created_at,
	created_by,
	updated_at,
	updated_by
) VALUES (
	'{{ $tbl.TableName }}',
	'{{ $tbl.Data.Label }}',
	'{{ $tbl.Data.Description }}',
	'{{ $tbl.Data.DisplayField }}',
	{{ $tbl.Data.Version }},
	to_timestamp('{{ $tbl.Data.CreatedAt }}', 'YYYY-MM-DDThh24:mi:ss.mssZ'),
	'system',
	to_timestamp('{{ $tbl.Data.UpdatedAt }}', 'YYYY-MM-DDThh24:mi:ss.mssZ'),
	'system'
)
ON CONFLICT (name) DO UPDATE
SET
	description = EXCLUDED.description,
	display_field = EXCLUDED.display_field,
	version = EXCLUDED.version,
	updated_at = EXCLUDED.updated_at,
	updated_by = EXCLUDED.updated_by
;
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}__meta (
	_id serial primary key,
	name text not null unique,
	label text not null,
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
CREATE UNIQUE INDEX IF NOT EXISTS name ON {{ $.SchemaName }}.{{ $tbl.TableName }}__meta(name);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}__meta_history (
	_id serial primary key,
	metaId integer not null,
	name text not null,
	fields jsonb not null,
	created_at timestamp without time zone default now(),
	created_by text not null
);
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}.on_{{ $tbl.TableName }}__meta_update()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}__meta_history (
		metaId,
		name,
		fields,
		created_by
	) VALUES (
		OLD._id,
		OLD.name,
		row_to_json(OLD),
		NEW.updated_by
	);
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}_{{ $tbl.TableName }}__meta_update ON {{ $.SchemaName }}.{{ $tbl.TableName }}__meta;
--
CREATE TRIGGER {{ $.SchemaName }}_{{ $tbl.TableName }}__meta_update
    AFTER UPDATE ON {{ $.SchemaName }}.{{ $tbl.TableName }}__meta
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on_{{ $tbl.TableName }}__meta_update();
--
{{ range $fieldsidx, $fields := $tbl.Data.Metas }}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}__meta (
	name,
	label,
	type,
	link_type,
	items,
	is_localized,
	is_required,
	is_disabled,
	is_omitted,
	validations,
	created_by,
	updated_by
) VALUES (
	'{{ .Name }}',
	'{{ .Label }}',
	'{{ .Type }}',
	'{{ .LinkType }}',
	{{ if .Items }}'{{ .Items }}'::jsonb{{ else }}NULL{{ end }},
	{{ .Localized }},
	{{ .Required }},
	{{ .Disabled }},
	{{ .Omitted }},
	'{{ if .Validations }}{{ .Validations }}{{ else }}[]{{ end }}'::jsonb,
	'system',
	'system'
)
ON CONFLICT (name) DO UPDATE
SET
	label = EXCLUDED.label,
	type = EXCLUDED.type,
	link_type = EXCLUDED.link_type,
	items = EXCLUDED.items,
	is_localized = EXCLUDED.is_localized,
	is_required = EXCLUDED.is_required,
	is_disabled = EXCLUDED.is_disabled,
	is_omitted = EXCLUDED.is_omitted,
	validations = EXCLUDED.validations,
	updated_at = now(),
	updated_by = EXCLUDED.updated_by
;
{{ end }}
--
{{ range $locidx, $loc := $.Space.Locales }}
{{$locale:=(fmtLocale $loc.Code)}}
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}_{{ $locale }} (
	_id serial primary key,
	sysId text not null unique,
	{{- range $colidx, $col := $tbl.Columns }}
	{{ .ColumnName }} {{ .ColumnType }}{{ .ColumnDesc }},
	{{- end }}
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS sysId ON {{ $.SchemaName }}.{{ $tbl.TableName }}_{{ $locale }}(sysId);
--
CREATE OR REPLACE FUNCTION {{ $tbl.TableName }}_{{ $locale }}_upsert(_sysId text,{{ range $colidx, $col := $tbl.Columns }} _{{ .ColumnName }} {{ .ColumnType }},{{ end }} _version integer, _created_at timestamp, _created_by text, _updated_at timestamp, _updated_by text)
RETURNS void AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}_{{ $locale }} (
	sysId,
	{{- range $colidx, $col := $tbl.Columns }}
	{{ .ColumnName }},
	{{- end }}
	version,
	created_at,
	created_by,
	updated_at,
	updated_by
) VALUES (
	_sysId,
	{{- range $colidx, $col := $tbl.Columns }}
	_{{ .ColumnName }},
	{{- end }}
	_version,
	_created_at,
	_created_by,
	_updated_at,
	_updated_by
)
ON CONFLICT (sysId) DO UPDATE
SET
	{{- range $colidx, $col := $tbl.Columns }}
	{{ .ColumnName }} = EXCLUDED.{{ .ColumnName }},
	{{- end }}
	version = EXCLUDED.version,
	updated_at = now(),
	updated_by = EXCLUDED.updated_by
;
END;
$$  LANGUAGE plpgsql;
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}_{{ $locale }}__publish (
	_id serial primary key,
	sysId text not null unique,
	{{- range $colidx, $col := $tbl.Columns }}
	{{ .ColumnName }} {{ .ColumnType }},
	{{- end }}
	version integer not null default 0,
	published_at timestamp without time zone default now(),
	published_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS sysId ON {{ $.SchemaName }}.{{ $tbl.TableName }}_{{ $locale }}__publish(sysId);
--
CREATE OR REPLACE FUNCTION {{ $tbl.TableName }}_{{ $locale }}_publish(_aid integer)
RETURNS integer AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}_{{ $locale }}__publish (
	sysId,
	{{- range $colidx, $col := $tbl.Columns }}
	{{ .ColumnName }},
	{{- end }}
	version,
	published_by
)
SELECT
	sysId,
	{{- range $colidx, $col := $tbl.Columns }}
	{{ .ColumnName }},
	{{- end }}
	version,
	updated_by
FROM {{ $.SchemaName }}.{{ $tbl.TableName }}_{{ $locale }}
WHERE _id = _aid
ON CONFLICT (sysId) DO UPDATE
SET
	{{- range $colidx, $col := $tbl.Columns }}
	{{ .ColumnName }} = EXCLUDED.{{ .ColumnName }},
	{{- end }}
	version = EXCLUDED.version,
	published_at = now(),
	published_by = EXCLUDED.published_by
;
END;
$$  LANGUAGE plpgsql;
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}_{{ $locale }}__history(
	_id serial primary key,
	pubId integer not null,
	sysId text not null,
	fields jsonb not null,
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null
);
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}.on_{{ $tbl.TableName }}_{{ $locale }}_update()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}_{{ $locale }}__history (
		pubId,
		sysId,
		fields,
		version,
		created_by
	) VALUES (
		OLD._id,
		OLD.sysId,
		row_to_json(OLD),
		OLD.version,
		NEW.updated_by
	);
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}_{{ $tbl.TableName }}_{{ $locale }}_update ON {{ $.SchemaName }}.{{ $tbl.TableName }}_{{ $locale }}__publish;
--
CREATE TRIGGER {{ $.SchemaName }}_{{ $tbl.TableName }}_{{ $locale }}_update
    AFTER UPDATE ON {{ $.SchemaName }}.{{ $tbl.TableName }}_{{ $locale }}__publish
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on_{{ $tbl.TableName }}_{{ $locale }}_update();
--
{{ end -}}
COMMIT;
{{ end -}}
BEGIN;
{{ range $locidx, $loc := $.Space.Locales }}
{{$locale:=(fmtLocale $loc.Code)}}
{{ range $refidx, $ref := $.References }}
ALTER TABLE {{ $.SchemaName }}.{{ .TableName }}_{{ $locale }}
  {{- range $colidx, $col := .Columns }}
  {{- if $colidx }},{{- end }}
  ADD COLUMN IF NOT EXISTS {{ .ColumnName }} integer references {{ $.SchemaName }}.{{ .ColumnDesc }}_{{ $locale }}(_id)
{{- end }};
ALTER TABLE {{ $.SchemaName }}.{{ .TableName }}_{{ $locale }}__publish
  {{- range $colidx, $col := .Columns }}
  {{- if $colidx }},{{- end }}
  ADD COLUMN IF NOT EXISTS {{ .ColumnName }} integer references {{ $.SchemaName }}.{{ .ColumnDesc }}_{{ $locale }}__publish(_id)
{{- end }};
{{ end -}}
{{ end -}}
COMMIT;`
