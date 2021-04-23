package gontentful

const pgFuncTemplate = `
CREATE SCHEMA IF NOT EXISTS {{ $.SchemaName }};
--
DROP TYPE IF EXISTS _filter CASCADE;
CREATE TYPE _filter AS (
	field TEXT,
	comparer TEXT,
	value TEXT
);
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = '_result') THEN
		CREATE TYPE _result AS (
			count INTEGER,
			items JSON
		);
	END IF;
END $$;
--
{{ range $i, $t := $.Tables }}
{{- if $.DropTables }}
DROP FUNCTION IF EXISTS _get_{{ .TableName }}_items CASCADE;
{{ end -}}
--
CREATE OR REPLACE FUNCTION _get_{{ .TableName }}_items(locale TEXT, filters TEXT[], orderBy TEXT, skip INTEGER, take INTEGER)
RETURNS TABLE(
	_id text,
	_sys_id text,
	_locale text,
	{{- range $j, $c:= .Columns }}
	{{- if $j -}},{{- end }}
	{{ if eq .ColumnName "limit" -}}_{{- end -}}
	{{- .ColumnName }} {{ .ColumnType }}
	{{- end }}
) AS $$
DECLARE
	qs text := '';
	filter text;
BEGIN
	qs:= 'SELECT _id,_sys_id,_locale,';
	
	qs:= qs || '{{- range $j, $c:= .Columns -}}
	{{- if $j -}},{{- end -}}
	{{- .ColumnName }} {{- if eq .ColumnName "limit" }} AS _ "{{ .ColumnName }}"{{- end -}}
	{{- end -}}';
	
	qs:= qs || ' FROM {{ .TableName }} WHERE ({{ .TableName }}._locale=''' || locale || ''')';

	IF filters IS NOT NULL THEN
		FOREACH filter IN ARRAY filters LOOP
			qs := qs || ' AND ({{ .TableName }}.' || filter || ')';
		END LOOP;
	END IF;

	IF orderBy <> '' THEN
	qs:= qs || ' ORDER BY ' || orderBy;
	END IF;

	IF skip <> 0 THEN
	qs:= qs || ' OFFSET ' || skip;
	END IF;

	IF take <> 0 THEN
	qs:= qs || ' LIMIT ' || take;
	END IF;

	RETURN QUERY EXECUTE qs;
END;
$$ LANGUAGE 'plpgsql';	
{{ end }}
--
CREATE OR REPLACE FUNCTION _get_sys_ids(tableName text, locale TEXT, filters TEXT[], orderBy TEXT, skip INTEGER, take INTEGER)
RETURNS SETOF text AS $$
DECLARE
	qs text := '';
	filter text;
BEGIN
	qs:= 'SELECT _sys_id FROM ' || tableName || ' WHERE (' || tableName || '._locale=''' || locale || ''')';

	IF filters IS NOT NULL THEN
		FOREACH filter IN ARRAY filters LOOP
			qs := qs || ' AND (' || tableName || '.' || filter || ')';
		END LOOP;
	END IF;

	IF orderBy <> '' THEN
	qs:= qs || ' ORDER BY ' || orderBy;
	END IF;

	IF skip <> 0 THEN
	qs:= qs || ' OFFSET ' || skip;
	END IF;

	IF take <> 0 THEN
	qs:= qs || ' LIMIT ' || take;
	END IF;

	RETURN QUERY EXECUTE qs;
END;
$$ LANGUAGE 'plpgsql';
--
{{- define "assetRef" -}}
(CASE WHEN {{ .Reference.JoinAlias }}._sys_id IS NULL THEN NULL ELSE json_build_object(
						'title', {{ .Reference.JoinAlias }}.title,
						'description', {{ .Reference.JoinAlias }}.description,
						'file', json_build_object(
							'contentType', {{ .Reference.JoinAlias }}.content_type,
							'fileName', {{ .Reference.JoinAlias }}.file_name,
							'url', {{ .Reference.JoinAlias }}.url
						)
					) END)
{{- end -}}
{{- define "assetCon" -}}
json_build_object('id', {{ .Reference.JoinAlias }}._sys_id) AS sys,
						{{ .Reference.JoinAlias }}.title AS "title",
						{{ .Reference.JoinAlias }}.description AS "description",
						json_build_object(
							'contentType', {{ .Reference.JoinAlias }}.content_type,
							'fileName', {{ .Reference.JoinAlias }}.file_name,
							'url', {{ .Reference.JoinAlias }}.url
						) AS "file"
{{- end -}}
{{- define "refColumn" -}} 
(CASE WHEN {{ .JoinAlias }}._sys_id IS NULL THEN NULL ELSE json_build_object(
					'sys', json_build_object('id', {{ .JoinAlias }}._sys_id)
					{{- range $i, $c:= .Columns -}}
					,
					'{{ .Alias }}',
					{{- if .ConTableName -}}
						_included_{{ .Reference.JoinAlias }}.res
					{{- else if .IsAsset -}}
						{{ template "assetRef" . }}	
					{{- else if .Reference -}}
						{{ template "refColumn" .Reference }}
					{{- else -}}
						{{ .JoinAlias }}.{{ .ColumnName }}
					{{- end -}}
					{{- end }}) END)
{{- end -}}
{{- define "conColumn" -}} 
json_build_object('id', {{ .JoinAlias }}._sys_id) AS sys
						{{- range $i, $c:= .Columns -}}
						,
						{{ if .ConTableName -}}
							_included_{{ .Reference.JoinAlias }}.res
						{{- else if .IsAsset -}}
						{{ template "assetRef" . }}
						{{- else if .Reference -}}
							{{ template "refColumn" .Reference }}
						{{- else -}}
							{{ .JoinAlias }}.{{ .ColumnName }}
						{{- end }} AS "{{ .Alias }}"
						{{- end }}
{{- end -}}
{{- define "join" -}}
		{{- if .ConTableName }}
			LEFT JOIN LATERAL (
				SELECT json_agg(l) AS res FROM (
					SELECT
						{{ if .IsAsset -}}
						{{ template "assetCon" . }}
						{{- else -}}
						{{ template "conColumn" .Reference }}
						{{- end }}
					FROM {{ .ConTableName }}
					JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }} ON {{ .Reference.JoinAlias }}._id = {{ .ConTableName }}.{{ .Reference.TableName }}
					{{- range .Reference.Columns }}
					{{- template "join" . }}
					{{- end }}
					WHERE {{ .ConTableName }}.{{ .TableName }} = {{ .JoinAlias }}._id 
					ORDER BY {{ .ConTableName }}._id 
				) l
			) _included_{{ .Reference.JoinAlias }} ON true
		{{- else if .Reference }}
			LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }} ON {{ .Reference.JoinAlias }}._id = {{ .JoinAlias }}.{{ .Reference.ForeignKey }}
			{{- range .Reference.Columns }}
			{{- template "join" . }}
			{{- end -}}
		{{- end -}}
{{- end -}}
{{ range $i, $t := $.Functions }}
CREATE OR REPLACE FUNCTION {{ .TableName }}_items(localeArg TEXT, filters TEXT[], orderBy TEXT, skip INTEGER, take INTEGER)
RETURNS json AS $$
BEGIN
	RETURN (
		WITH filtered AS (
			SELECT * FROM _get_{{- .TableName -}}_items(localeArg, filters, orderBy, skip, take)
		)
		SELECT json_agg(t) AS res FROM (
			SELECT
				json_build_object('id', {{ .TableName }}._sys_id) AS sys
			{{- range .Columns -}}
				,
				{{ if .ConTableName -}}
					_included_{{ .Reference.JoinAlias }}.res
				{{- else if .IsAsset -}}
					{{ template "assetRef" . }}
				{{- else if .Reference -}}
					{{ template "refColumn" .Reference }}
				{{- else -}}
					{{ .TableName }}.{{ .ColumnName }}
				{{- end }} AS "{{ .Alias }}"
			{{- end }}
			FROM filtered {{ .TableName }}
			{{- range .Columns -}}
				{{ template "join" . }}
			{{- end }}
			WHERE {{ .TableName }}._locale = localeArg
		) t
	);
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ .TableName }}_query(localeArg TEXT, filters TEXT[], orderBy TEXT, skip INTEGER, take INTEGER)
RETURNS _result AS $$
DECLARE
	count integer;
	items json;
	res _result;
BEGIN
	SELECT COUNT(f) FROM (SELECT _get_sys_ids('{{ .TableName }}', localeArg, filters, '', 0, 0)) AS f INTO count;
	SELECT {{ .TableName }}_items(localeArg, filters, orderBy, skip, take) INTO items;
	IF items IS NULL THEN
		items:= '[]'::JSON;
	END IF;
	RETURN ROW(count, items)::_result;
END;
$$ LANGUAGE 'plpgsql';
--
{{- end }}
{{- range $i, $t := $.DeleteTriggers }}
CREATE OR REPLACE FUNCTION {{ .TableName }}_delete_trigger() 
   RETURNS TRIGGER 
AS $$
BEGIN
	{{- range $idx, $c := .ConTables }}
	DELETE FROM {{ $.SchemaName }}.{{ . }} where {{ . }}.{{ $t.TableName }} = OLD._id;
	{{- end }}
	RETURN OLD;
END
$$ LANGUAGE 'plpgsql';
--
DROP TRIGGER IF EXISTS {{ .TableName }}_delete
ON {{ .TableName }};
--
CREATE TRIGGER {{ .TableName }}_delete 
	AFTER DELETE 
ON {{ .TableName }} 
FOR EACH ROW 
	EXECUTE PROCEDURE {{ .TableName }}_delete_trigger();
--
{{- end }}
`
