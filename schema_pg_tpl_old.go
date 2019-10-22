package gontentful

const pgTemplateOld = `
CREATE SCHEMA IF NOT EXISTS {{ $.SchemaName }};
--
CREATE TABLE IF NOT EXISTS _space (
	_id serial primary key,
	spaceid text not null unique,
	name text not null,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null
);
CREATE UNIQUE INDEX IF NOT EXISTS spaceid ON _space(spaceid);
--
CREATE TABLE IF NOT EXISTS _locales (
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
CREATE UNIQUE INDEX IF NOT EXISTS code ON _locales(code);
--
CREATE TABLE IF NOT EXISTS _models (
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
CREATE UNIQUE INDEX IF NOT EXISTS name ON _models(name);
--
CREATE TABLE IF NOT EXISTS _entries (
	_id serial primary key,
	sys_id text not null unique,
	table_name text not null
);
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON _entries(sys_id);
--
{{ range $locidx, $loc := $.Space.Locales }}
{{$locale:=(fmtLocale $loc.Code)}}
INSERT INTO _locales (
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
CREATE TABLE IF NOT EXISTS _asset$meta (
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
CREATE UNIQUE INDEX IF NOT EXISTS name ON _asset$meta(name);
--
{{ range $aidx, $col := $.AssetColumns }}
INSERT INTO _asset$meta (
	name,
	label,
	type,
	created_by,
	updated_by
) VALUES (
	'{{ $col }}',
	'{{ $col }}',
	'Text',
	'system',
	'system'
)
ON CONFLICT (name) DO NOTHING;
{{- end -}}
--
CREATE TABLE IF NOT EXISTS _asset${{ $locale }} (
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
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON _asset${{ $locale }}(sys_id);
--
DROP FUNCTION IF EXISTS on$asset${{ $locale }}_insert() CASCADE;
--
CREATE FUNCTION on$asset${{ $locale }}_insert()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO _entries (
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
DROP TRIGGER IF EXISTS $asset${{ $locale }}_insert ON _asset${{ $locale }};
--
CREATE TRIGGER $asset${{ $locale }}_insert
	AFTER INSERT ON _asset${{ $locale }}
	FOR EACH ROW
	EXECUTE PROCEDURE on$asset${{ $locale }}_insert();
--
DROP FUNCTION IF EXISTS on$asset${{ $locale }}_delete() CASCADE;
--
CREATE FUNCTION on$asset${{ $locale }}_delete()
RETURNS TRIGGER AS $$
BEGIN
	DELETE FROM _entries WHERE sys_id = OLD.sys_id AND table_name = '_asset${{ $locale }}';
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS $asset${{ $locale }}_delete ON _asset${{ $locale }};
--
CREATE TRIGGER $asset${{ $locale }}_delete
	AFTER DELETE ON _asset${{ $locale }}
	FOR EACH ROW
	EXECUTE PROCEDURE on$asset${{ $locale }}_delete();
--
{{ end -}}
----
{{ range $tblidx, $tbl := $.Tables }}
INSERT INTO _models (
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
CREATE TABLE IF NOT EXISTS {{ $tbl.TableName }}$meta (
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
CREATE UNIQUE INDEX IF NOT EXISTS name ON {{ $tbl.TableName }}$meta(name);
--
{{ range $fieldsidx, $fields := $tbl.Data.Metas }}
INSERT INTO {{ $tbl.TableName }}$meta (
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
CREATE TABLE IF NOT EXISTS {{ $tbl.TableName }}${{ $locale }} (
	_id serial primary key,
	sys_id text not null unique,
	{{- range $colidx, $col := $tbl.Columns }}
	"{{ .ColumnName }}" {{ .ColumnType }},
	{{- end }}
	version integer not null default 0,
	created_at timestamp without time zone not null default now(),
	created_by text not null,
	updated_at timestamp without time zone not null default now(),
	updated_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON {{ $tbl.TableName }}${{ $locale }}(sys_id);
--
DROP FUNCTION IF EXISTS on_{{ $tbl.TableName }}${{ $locale }}_insert() CASCADE;
--
CREATE FUNCTION on_{{ $tbl.TableName }}${{ $locale }}_insert()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO _entries (
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
DROP TRIGGER IF EXISTS _{{ $tbl.TableName }}${{ $locale }}_insert ON {{ $tbl.TableName }}${{ $locale }};
--
CREATE TRIGGER _{{ $tbl.TableName }}${{ $locale }}_insert
    AFTER INSERT ON {{ $tbl.TableName }}${{ $locale }}
    FOR EACH ROW
	EXECUTE PROCEDURE on_{{ $tbl.TableName }}${{ $locale }}_insert();
--
DROP FUNCTION IF EXISTS on_{{ $tbl.TableName }}${{ $locale }}_delete() CASCADE;
--
CREATE FUNCTION on_{{ $tbl.TableName }}${{ $locale }}_delete()
RETURNS TRIGGER AS $$
BEGIN
	DELETE FROM _entries WHERE sys_id = OLD.sys_id AND table_name = '{{ $tbl.TableName }}${{ $locale }}';
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS _{{ $tbl.TableName }}${{ $locale }}_delete ON {{ $tbl.TableName }}${{ $locale }};
--
CREATE TRIGGER _{{ $tbl.TableName }}${{ $locale }}_delete
	AFTER DELETE ON {{ $tbl.TableName }}${{ $locale }}
	FOR EACH ROW
	EXECUTE PROCEDURE on_{{ $tbl.TableName }}${{ $locale }}_delete();
--
{{ end -}}
{{ end -}}
`
