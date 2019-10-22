package gontentful

const pgTriggers = `
BEGIN;
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on$models_update() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on$models_update()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}._models$history (
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
DROP TRIGGER IF EXISTS {{ $.SchemaName }}$models_update ON {{ $.SchemaName }}._models;
--
CREATE TRIGGER {{ $.SchemaName }}$models_update
    AFTER UPDATE ON {{ $.SchemaName }}._models
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on$models_update();
--
{{ range $locidx, $loc := $.Space.Locales }}
{{$locale:=(fmtLocale $loc.Code)}}
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.asset${{ $locale }}_upsert(text, text, text, text, text, text, integer, timestamp, text, timestamp, text) CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.asset${{ $locale }}_upsert(_sysId text, _title text, _description text, _fileName text, _contentType text, _url text, _version integer, _created_at timestamp, _created_by text, _updated_at timestamp, _updated_by text)
RETURNS void AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}._asset${{ $locale }} (
	sys_id,
	title,
	description,
	filename,
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
	filename = EXCLUDED.filename,
	content_type = EXCLUDED.content_type,
	url = EXCLUDED.url,
	version = EXCLUDED.version,
	updated_at = now(),
	updated_by = EXCLUDED.updated_by
;
END;
$$  LANGUAGE plpgsql;
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on$asset${{ $locale }}_insert() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on$asset${{ $locale }}_insert()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}._entries (
		sys_id,
		table_name
	) VALUES (
		NEW.sys_id,
		'_asset${{ $locale }}'
	) ON CONFLICT (sys_id) DO NOTHING;
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}$asset${{ $locale }}_insert ON {{ $.SchemaName }}._asset${{ $locale }};
--
CREATE TRIGGER {{ $.SchemaName }}$asset${{ $locale }}_insert
	AFTER INSERT ON {{ $.SchemaName }}._asset${{ $locale }}
	FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on$asset${{ $locale }}_insert();
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on$asset${{ $locale }}_delete() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on$asset${{ $locale }}_delete()
RETURNS TRIGGER AS $$
BEGIN
	DELETE FROM {{ $.SchemaName }}._entries WHERE sys_id = OLD.sys_id AND table_name = '_asset${{ $locale }}';
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}$asset${{ $locale }}_delete ON {{ $.SchemaName }}._asset${{ $locale }};
--
CREATE TRIGGER {{ $.SchemaName }}$asset${{ $locale }}_delete
	AFTER DELETE ON {{ $.SchemaName }}._asset${{ $locale }}
	FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on$asset${{ $locale }}_delete();
--
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.asset${{ $locale }}_publish(integer) CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.asset${{ $locale }}_publish(_aid integer)
RETURNS void AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}._asset${{ $locale }}$publish (
	sys_id,
	title,
	description,
	filename,
	content_type,
	url,
	version,
	published_by
)
SELECT
	sys_id,
	title,
	description,
	filename,
	content_type,
	url,
	version,
	updated_by
FROM {{ $.SchemaName }}._asset${{ $locale }}
WHERE _id = _aid
ON CONFLICT (sys_id) DO UPDATE
SET
	title = EXCLUDED.title,
	description = EXCLUDED.description,
	filename = EXCLUDED.filename,
	content_type = EXCLUDED.content_type,
	url = EXCLUDED.url,
	version = EXCLUDED.version,
	published_at = now(),
	published_by = EXCLUDED.published_by
;
END;
$$  LANGUAGE plpgsql;
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._asset${{ $locale }}$history(
	_id serial primary key,
	pub_id integer not null,
	sys_id text not null,
	fields jsonb not null,
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null
);
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on$asset${{ $locale }}$publish_update() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on$asset${{ $locale }}$publish_update()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}._asset${{ $locale }}$history (
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
DROP TRIGGER IF EXISTS {{ $.SchemaName }}$asset${{ $locale }}_update ON {{ $.SchemaName }}._asset${{ $locale }}$publish;
--
CREATE TRIGGER {{ $.SchemaName }}$asset${{ $locale }}$publish_update
    AFTER UPDATE ON {{ $.SchemaName }}._asset${{ $locale }}$publish
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on$asset${{ $locale }}$publish_update();
--
{{ end -}}
COMMIT;
--
{{ range $tblidx, $tbl := $.Tables }}
BEGIN;
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on_{{ $tbl.TableName }}$meta_update() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on_{{ $tbl.TableName }}$meta_update()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}$meta_history (
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
DROP TRIGGER IF EXISTS {{ $.SchemaName }}_{{ $tbl.TableName }}$meta_update ON {{ $.SchemaName }}.{{ $tbl.TableName }}$meta;
--
CREATE TRIGGER {{ $.SchemaName }}_{{ $tbl.TableName }}$meta_update
    AFTER UPDATE ON {{ $.SchemaName }}.{{ $tbl.TableName }}$meta
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on_{{ $tbl.TableName }}$meta_update();
--
{{ range $locidx, $loc := $.Space.Locales }}
{{$locale:=(fmtLocale $loc.Code)}}
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}_upsert(text,{{ range $colidx, $col := $tbl.Columns }} {{ .ColumnType }},{{ end }} integer, timestamp, text, timestamp, text) CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}_upsert(_sysId text,{{ range $colidx, $col := $tbl.Columns }} _{{ .ColumnName }} {{ .ColumnType }},{{ end }} _version integer, _created_at timestamp, _created_by text, _updated_at timestamp, _updated_by text)
RETURNS void AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }} (
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
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on_{{ $tbl.TableName }}${{ $locale }}_insert() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on_{{ $tbl.TableName }}${{ $locale }}_insert()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}._entries (
		sys_id,
		table_name
	) VALUES (
		NEW.sys_id,
		'{{ $tbl.TableName }}${{ $locale }}'
	) ON CONFLICT (sys_id) DO NOTHING;
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}_{{ $tbl.TableName }}${{ $locale }}_insert ON {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }};
--
CREATE TRIGGER {{ $.SchemaName }}_{{ $tbl.TableName }}${{ $locale }}_insert
    AFTER INSERT ON {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on_{{ $tbl.TableName }}${{ $locale }}_insert();
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on_{{ $tbl.TableName }}${{ $locale }}_delete() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on_{{ $tbl.TableName }}${{ $locale }}_delete()
RETURNS TRIGGER AS $$
BEGIN
	DELETE FROM {{ $.SchemaName }}._entries WHERE sys_id = OLD.sys_id AND table_name = '{{ $tbl.TableName }}${{ $locale }}';
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}_{{ $tbl.TableName }}${{ $locale }}_delete ON {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }};
--
CREATE TRIGGER {{ $.SchemaName }}_{{ $tbl.TableName }}${{ $locale }}_delete
	AFTER DELETE ON {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}
	FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on_{{ $tbl.TableName }}${{ $locale }}_delete();
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}_publish(integer) CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}_publish(_aid integer)
RETURNS integer AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}$publish (
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
FROM {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}$history(
	_id serial primary key,
	pub_id integer not null,
	sys_id text not null,
	fields jsonb not null,
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null
);
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on_{{ $tbl.TableName }}${{ $locale }}$publish_update() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on_{{ $tbl.TableName }}${{ $locale }}$publish_update()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}$history (
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
DROP TRIGGER IF EXISTS {{ $.SchemaName }}_{{ $tbl.TableName }}${{ $locale }}$publish_update ON {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}$publish;
--
CREATE TRIGGER {{ $.SchemaName }}_{{ $tbl.TableName }}${{ $locale }}$publish_update
    AFTER UPDATE ON {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}$publish
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on_{{ $tbl.TableName }}${{ $locale }}$publish_update();
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on_{{ $tbl.TableName }}${{ $locale }}$publish_delete() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on_{{ $tbl.TableName }}${{ $locale }}$publish_delete()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}$history (
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
DROP TRIGGER IF EXISTS {{ $.SchemaName }}_{{ $tbl.TableName }}${{ $locale }}$publish_delete ON {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}$publish;
--
CREATE TRIGGER {{ $.SchemaName }}_{{ $tbl.TableName }}${{ $locale }}$publish_delete
    AFTER DELETE ON {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}$publish
    FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on_{{ $tbl.TableName }}${{ $locale }}$publish_delete();
--
{{ end -}}
COMMIT;
{{ end -}}
`
