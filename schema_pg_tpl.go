package gontentful

const pgTemplate = `BEGIN;
{{ if .Drop }}
DROP SCHEMA IF EXISTS {{ $.SchemaName }} CASCADE;
{{ end -}}
CREATE SCHEMA IF NOT EXISTS {{ $.SchemaName }};
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._space (
	_id serial primary key,
	spaceid text not null unique,
	name text not null,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null
);
CREATE UNIQUE INDEX IF NOT EXISTS spaceid ON {{ $.SchemaName }}._space(spaceid);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._locales (
	_id serial primary key,
	code text not null unique,
	name text not null,
	is_default boolean,
	fallback_code text,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null
);
CREATE UNIQUE INDEX IF NOT EXISTS code ON {{ $.SchemaName }}._locales(code);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._models (
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
CREATE UNIQUE INDEX IF NOT EXISTS name ON {{ $.SchemaName }}._models(name);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._models__history(
	_id serial primary key,
	pub_id integer not null,
	name text not null,
	fields jsonb not null,
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null
);
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on__models_update() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on__models_update()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}._models__history (
		pub_id,
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._entries (
	_id serial primary key,
	sys_id text not null unique,
	table_name text not null
);
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON {{ $.SchemaName }}._entries(sys_id);
--
{{ range $locidx, $loc := $.Space.Locales }}
{{$locale:=(fmtLocale $loc.Code)}}
INSERT INTO {{ $.SchemaName }}._locales (
	code,
	name,
	is_default,
	fallback_code,
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
	is_default = EXCLUDED.is_default,
	fallback_code = EXCLUDED.fallback_code,
	updated_at = EXCLUDED.updated_at,
	updated_by = EXCLUDED.updated_by
;
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._assets__{{ $locale }} (
	_id serial primary key,
	sys_id text not null unique,
	title text not null,
	description text,
	file_name text,
	content_type text,
	url text,
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON {{ $.SchemaName }}._assets__{{ $locale }}(sys_id);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._assets__{{ $locale }}__publish (
	_id serial primary key,
	sys_id text not null unique,
	title text not null,
	description text,
	file_name text,
	content_type text,
	url text,
	version integer not null default 0,
	published_at timestamp without time zone default now(),
	published_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON {{ $.SchemaName }}._assets__{{ $locale }}__publish(sys_id);
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.assets__{{ $locale }}_upsert(text, text, text, text, text, text, integer, timestamp, text, timestamp, text) CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.assets__{{ $locale }}_upsert(_sysId text, _title text, _description text, _fileName text, _contentType text, _url text, _version integer, _created_at timestamp, _created_by text, _updated_at timestamp, _updated_by text)
RETURNS void AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}._assets__{{ $locale }} (
	sys_id,
	title,
	description,
	file_name,
	content_type,
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
ON CONFLICT (sys_id) DO UPDATE
SET
	title = EXCLUDED.title,
	description = EXCLUDED.description,
	file_name = EXCLUDED.file_name,
	content_type = EXCLUDED.content_type,
	url = EXCLUDED.url,
	version = EXCLUDED.version,
	updated_at = now(),
	updated_by = EXCLUDED.updated_by
;
END;
$$  LANGUAGE plpgsql;
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on__assets__{{ $locale }}_insert() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on__assets__{{ $locale }}_insert()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}._entries (
		sys_id,
		table_name
	) VALUES (
		NEW.sys_id,
		'_assets__{{ $locale }}'
	) ON CONFLICT (sys_id) DO NOTHING;
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}__assets__{{ $locale }}_insert ON {{ $.SchemaName }}._assets__{{ $locale }};
--
CREATE TRIGGER {{ $.SchemaName }}__assets__{{ $locale }}_insert
	AFTER INSERT ON {{ $.SchemaName }}._assets__{{ $locale }}
	FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on__assets__{{ $locale }}_insert();
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on__assets__{{ $locale }}_delete() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on__assets__{{ $locale }}_delete()
RETURNS TRIGGER AS $$
BEGIN
	DELETE FROM {{ $.SchemaName }}._entries WHERE sys_id = OLD.sys_id AND table_name = '_assets__{{ $locale }}';
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}__assets__{{ $locale }}_delete ON {{ $.SchemaName }}._assets__{{ $locale }};
--
CREATE TRIGGER {{ $.SchemaName }}__assets__{{ $locale }}_delete
	AFTER DELETE ON {{ $.SchemaName }}._assets__{{ $locale }}
	FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on__assets__{{ $locale }}_delete();
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.assets__{{ $locale }}_publish(integer) CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.assets__{{ $locale }}_publish(_aid integer)
RETURNS void AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}._assets__{{ $locale }}__publish (
	sys_id,
	title,
	description,
	file_name,
	content_type,
	url,
	version,
	published_by
)
SELECT
	sys_id,
	title,
	description,
	file_name,
	content_type,
	url,
	version,
	updated_by
FROM {{ $.SchemaName }}._assets__{{ $locale }}
WHERE _id = _aid
ON CONFLICT (sys_id) DO UPDATE
SET
	title = EXCLUDED.title,
	description = EXCLUDED.description,
	file_name = EXCLUDED.file_name,
	content_type = EXCLUDED.content_type,
	url = EXCLUDED.url,
	version = EXCLUDED.version,
	published_at = now(),
	published_by = EXCLUDED.published_by
;
END;
$$  LANGUAGE plpgsql;
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._assets__{{ $locale }}__history(
	_id serial primary key,
	pub_id integer not null,
	sys_id text not null,
	fields jsonb not null,
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null
);
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on__assets__{{ $locale }}__publish_update() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on__assets__{{ $locale }}__publish_update()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}._assets__{{ $locale }}__history (
		pub_id,
		sys_id,
		fields,
		version,
		created_by
	) VALUES (
		OLD._id,
		OLD.sys_id,
		row_to_json(OLD),
		OLD.version,
		NEW.published_by
	);
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}__assets__{{ $locale }}_update ON {{ $.SchemaName }}._assets__{{ $locale }}__publish;
--
CREATE TRIGGER {{ $.SchemaName }}__assets__{{ $locale }}__publish_update
    AFTER UPDATE ON {{ $.SchemaName }}._assets__{{ $locale }}__publish
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on__assets__{{ $locale }}__publish_update();
--
{{ end -}}
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}___meta (
	_id serial primary key,
	name text not null unique,
	label text not null,
	type text not null,
	items_type text,
	link_type text,
	is_localized boolean default false,
	is_required boolean default false,
	is_unique boolean default false,
	is_disabled boolean default false,
	is_omitted boolean default false,
	created_at timestamp without time zone not null default now(),
	created_by text not null,
	updated_at timestamp without time zone not null default now(),
	updated_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS name ON {{ $.SchemaName }}.{{ $tbl.TableName }}___meta(name);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}___meta_history (
	_id serial primary key,
	meta_id integer not null,
	name text not null,
	fields jsonb not null,
	created_at timestamp without time zone default now(),
	created_by text not null
);
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on_{{ $tbl.TableName }}___meta_update() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on_{{ $tbl.TableName }}___meta_update()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}___meta_history (
		meta_id,
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
DROP TRIGGER IF EXISTS {{ $.SchemaName }}_{{ $tbl.TableName }}___meta_update ON {{ $.SchemaName }}.{{ $tbl.TableName }}___meta;
--
CREATE TRIGGER {{ $.SchemaName }}_{{ $tbl.TableName }}___meta_update
    AFTER UPDATE ON {{ $.SchemaName }}.{{ $tbl.TableName }}___meta
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on_{{ $tbl.TableName }}___meta_update();
--
{{ range $fieldsidx, $fields := $tbl.Data.Metas }}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}___meta (
	name,
	label,
	type,
	items_type,
	link_type,
	is_localized,
	is_required,
	is_unique,
	is_disabled,
	is_omitted,
	created_by,
	updated_by
) VALUES (
	'{{ .Name }}',
	'{{ .Label }}',
	'{{ .Type }}',
	'{{ .ItemsType }}',
	'{{ .LinkType }}',
	{{ .Localized }},
	{{ .Required }},
	{{ .Unique }},
	{{ .Disabled }},
	{{ .Omitted }},
	'system',
	'system'
)
ON CONFLICT (name) DO UPDATE
SET
	label = EXCLUDED.label,
	type = EXCLUDED.type,
	items_type = EXCLUDED.items_type,
	link_type = EXCLUDED.link_type,
	is_localized = EXCLUDED.is_localized,
	is_required = EXCLUDED.is_required,
	is_unique = EXCLUDED.is_unique,
	is_disabled = EXCLUDED.is_disabled,
	is_omitted = EXCLUDED.is_omitted,
	updated_at = now(),
	updated_by = EXCLUDED.updated_by
;
{{ end }}
--
{{ range $locidx, $loc := $.Space.Locales }}
{{$locale:=(fmtLocale $loc.Code)}}
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }} (
	_id serial primary key,
	sys_id text not null unique,
	{{- range $colidx, $col := $tbl.Columns }}
	{{ .ColumnName }} {{ .ColumnType }},
	{{- end }}
	version integer not null default 0,
	created_at timestamp without time zone not null default now(),
	created_by text not null,
	updated_at timestamp without time zone not null default now(),
	updated_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}(sys_id);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}__publish (
	_id serial primary key,
	sys_id text not null unique,
	{{- range $colidx, $col := $tbl.Columns }}
	{{ .ColumnName }} {{ .ColumnType }}{{ .ColumnDesc }}{{- if and .Required (eq $locale $.DefaultLocale) }} not null{{- end -}},
	{{- end }}
	version integer not null default 0,
	published_at timestamp without time zone not null default now(),
	published_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}__publish(sys_id);
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}_upsert(text,{{ range $colidx, $col := $tbl.Columns }} {{ .ColumnType }},{{ end }} integer, timestamp, text, timestamp, text) CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}_upsert(_sysId text,{{ range $colidx, $col := $tbl.Columns }} _{{ .ColumnName }} {{ .ColumnType }},{{ end }} _version integer, _created_at timestamp, _created_by text, _updated_at timestamp, _updated_by text)
RETURNS void AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }} (
	sys_id,
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
ON CONFLICT (sys_id) DO UPDATE
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
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on_{{ $tbl.TableName }}__{{ $locale }}_insert() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on_{{ $tbl.TableName }}__{{ $locale }}_insert()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}._entries (
		sys_id,
		table_name
	) VALUES (
		NEW.sys_id,
		'{{ $tbl.TableName }}__{{ $locale }}'
	) ON CONFLICT (sys_id) DO NOTHING;
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}_{{ $tbl.TableName }}__{{ $locale }}_insert ON {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }};
--
CREATE TRIGGER {{ $.SchemaName }}_{{ $tbl.TableName }}__{{ $locale }}_insert
    AFTER INSERT ON {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on_{{ $tbl.TableName }}__{{ $locale }}_insert();
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on_{{ $tbl.TableName }}__{{ $locale }}_delete() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on_{{ $tbl.TableName }}__{{ $locale }}_delete()
RETURNS TRIGGER AS $$
BEGIN
	DELETE FROM {{ $.SchemaName }}._entries WHERE sys_id = OLD.sys_id AND table_name = '{{ $tbl.TableName }}__{{ $locale }}';
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}_{{ $tbl.TableName }}__{{ $locale }}_delete ON {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }};
--
CREATE TRIGGER {{ $.SchemaName }}_{{ $tbl.TableName }}__{{ $locale }}_delete
	AFTER DELETE ON {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}
	FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on_{{ $tbl.TableName }}__{{ $locale }}_delete();
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}_publish(integer) CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}_publish(_aid integer)
RETURNS integer AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}__publish (
	sys_id,
	{{- range $colidx, $col := $tbl.Columns }}
	{{ .ColumnName }},
	{{- end }}
	version,
	published_by
)
SELECT
	sys_id,
	{{- range $colidx, $col := $tbl.Columns }}
	{{ .ColumnName }},
	{{- end }}
	version,
	updated_by
FROM {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}
WHERE _id = _aid
ON CONFLICT (sys_id) DO UPDATE
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}__history(
	_id serial primary key,
	pub_id integer not null,
	sys_id text not null,
	fields jsonb not null,
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null
);
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on_{{ $tbl.TableName }}__{{ $locale }}__publish_update() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on_{{ $tbl.TableName }}__{{ $locale }}__publish_update()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}__history (
		pub_id,
		sys_id,
		fields,
		version,
		created_by
	) VALUES (
		OLD._id,
		OLD.sys_id,
		row_to_json(OLD),
		OLD.version,
		NEW.published_by
	);
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}_{{ $tbl.TableName }}__{{ $locale }}__publish_update ON {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}__publish;
--
CREATE TRIGGER {{ $.SchemaName }}_{{ $tbl.TableName }}__{{ $locale }}__publish_update
    AFTER UPDATE ON {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}__publish
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on_{{ $tbl.TableName }}__{{ $locale }}__publish_update();
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on_{{ $tbl.TableName }}__{{ $locale }}__publish_delete() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on_{{ $tbl.TableName }}__{{ $locale }}__publish_delete()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}__history (
		pub_id,
		sys_id,
		fields,
		version,
		created_by
	) VALUES (
		OLD._id,
		OLD.sys_id,
		row_to_json(OLD),
		OLD.version,
		'sync'
	);
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}_{{ $tbl.TableName }}__{{ $locale }}__publish_delete ON {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}__publish;
--
CREATE TRIGGER {{ $.SchemaName }}_{{ $tbl.TableName }}__{{ $locale }}__publish_delete
    AFTER DELETE ON {{ $.SchemaName }}.{{ $tbl.TableName }}__{{ $locale }}__publish
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on_{{ $tbl.TableName }}__{{ $locale }}__publish_delete();
--
{{ end -}}
{{ end -}}
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._get_columns(tableName text, locale text, defaultLocale text, usePreview boolean)
RETURNS text AS $$
DECLARE
	qs text;
	suffix text := '__publish';
	isFirst boolean := true;
	meta record;
BEGIN
	IF usePreview THEN
		suffix := '';
	END IF;

	qs := 'SELECT ';
	FOR meta IN
		EXECUTE 'SELECT
		name,
		is_localized
        FROM content.' || tableName || '___meta' LOOP

	    IF isFirst THEN
	    	isFirst := false;
	    ELSE
	    	qs := qs || ', ';
	    END IF;

		IF meta.is_localized AND locale <> defaultLocale THEN
			qs := 'COALESCE(' || tableName || '_' || locale || '.' || meta.name || ',' ||
			tableName || '_' || defaultLocale || '.' || meta.name || ')';
		ELSE
	    	qs := qs || tableName || '_' || defaultLocale || '.' || meta.name;
		END IF;

		qs := qs || ' as ' || meta.name;
    END LOOP;

	qs := qs || ' FROM content.' || tableName || '_' || defaultLocale || suffix || ' ' || tableName || '_' || defaultLocale;

	IF locale <> defaultLocale THEN
		qs := qs || ' LEFT JOIN content.' || tableName || '_' || locale || '__publish ' || tableName || '_' || locale ||
		' ON ' || tableName || '_' || defaultLocale || '.sys_id = ' || tableName || '_' || locale || '.sys_id';
	END IF;

	RETURN qs;
END;
$$ LANGUAGE 'plpgsql';
COMMIT;
`
