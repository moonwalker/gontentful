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
DROP TYPE IF EXISTS _result CASCADE;
CREATE TYPE _result AS (
	count INTEGER,
	items JSON
);
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
{{- define "asset" -}}
json_build_object(
						'title', {{ .Reference.JoinAlias }}.title,
						'description', {{ .Reference.JoinAlias }}.description,
						'file', json_build_object(
							'contentType', {{ .Reference.JoinAlias }}.content_type,
							'fileName', {{ .Reference.JoinAlias }}.file_name,
							'url', {{ .Reference.JoinAlias }}.url
						)
					)
{{- end -}}
{{- define "refColumn" -}} 
json_build_object(
					'sys', json_build_object('id', {{ .JoinAlias }}._sys_id),
					{{- range $i, $c:= .Columns }}
					{{- if $i -}},{{- end }}
					'{{ .Alias }}',
					{{- if .IsAsset -}}
					(CASE WHEN {{ .JoinAlias }}.{{ .ColumnName }} IS NULL THEN NULL ELSE {{ template "asset" . }} END)
					{{- else if .ConTableName -}}
						_included_{{ .Reference.JoinAlias }}.res
					{{- else if .Reference -}}
						(CASE WHEN {{ .JoinAlias }}.{{ .ColumnName }} IS NULL THEN NULL ELSE {{ template "refColumn" .Reference }} END)
					{{- else -}}
						{{ .JoinAlias }}.{{ .ColumnName }}
					{{- end -}}
					{{- end }})
{{- end -}}
{{- define "join" -}}
		{{- if .ConTableName }}
			LEFT JOIN LATERAL (
				SELECT json_agg(l) AS res FROM (
					SELECT
						json_build_object('id', {{ .Reference.JoinAlias }}._sys_id) AS sys
						{{- range $i, $c:= .Reference.Columns -}}
							,
							{{ .JoinAlias }}.{{ .ColumnName }} AS "{{ .Alias }}"
						{{- end }}
					FROM {{ .ConTableName }}
					JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }} ON {{ .Reference.JoinAlias }}._sys_id = {{ .ConTableName }}.{{ .Reference.TableName }} AND {{ .Reference.JoinAlias }}._locale = localeArg
					WHERE {{ .ConTableName }}.{{ .TableName }} = {{ .JoinAlias }}._sys_id AND {{ .ConTableName }}._locale = localeArg
				) l
			) _included_{{ .Reference.JoinAlias }} ON true
		{{- else if .Reference }}
			LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }} ON {{ .Reference.JoinAlias }}._sys_id = {{ .JoinAlias }}.{{ .Reference.ForeignKey }} AND {{ .Reference.JoinAlias }}._locale = localeArg
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
			SELECT _get_sys_ids('{{ .TableName }}', localeArg, filters, orderBy, skip, take) AS _sys_id
		)
		SELECT json_agg(t) AS res FROM (
			SELECT
				json_build_object('id', {{ .TableName }}._sys_id) AS sys
			{{- range .Columns -}}
				,
				{{ if .IsAsset -}}
					{{ template "asset" . }}
				{{- else if .ConTableName -}}
					_included_{{ .Reference.JoinAlias }}.res
				{{- else if .Reference -}}
					(CASE WHEN {{ .JoinAlias }}.{{ .ColumnName }} IS NULL THEN NULL ELSE {{ template "refColumn" .Reference }} END)
				{{- else -}}
					{{ .TableName }}.{{ .ColumnName }}
				{{- end }} AS "{{ .Alias }}"
			{{- end }}
			FROM {{ .TableName }}
			{{- range .Columns -}}
				{{ template "join" . }}
			{{- end }}
			WHERE {{ .TableName }}._locale = localeArg AND {{ .TableName }}._sys_id IN (SELECT _sys_id FROM filtered)
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
{{- end -}}
`
