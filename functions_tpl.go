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
CREATE OR REPLACE FUNCTION _get_sys_ids(tableName text, locale TEXT, filters _filter[], orderBy TEXT, skip INTEGER, take INTEGER)
RETURNS SETOF text AS $$
DECLARE
	qs text := '';
	filter _filter;
BEGIN
	qs:= 'SELECT _sys_id FROM ' || tableName || ' WHERE (' || tableName || '._locale=''' || locale || ''')';

	IF filters IS NOT NULL THEN
		FOREACH filter IN ARRAY filters LOOP
			qs := qs || ' AND (' || tableName || '.' || filter.field || filter.comparer || filter.value || ')';
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
	'title', {{ .ColumnName }}__asset.title,
	'description', {{ .ColumnName }}__asset.description,
	'file', json_build_object(
		'contentType', {{ .ColumnName }}__asset.content_type,
		'fileName', {{ .ColumnName }}__asset.file_name,
		'url', {{ .ColumnName }}__asset.url
	)
)
{{- end -}}
{{- define "refColumn" -}}
	json_build_object(
		'sys', json_build_object('id', {{ .TableName }}._sys_id),
	{{- range $i, $c:= .Columns }}
		{{- if $i -}},{{- end }}
		'{{ .Alias }}',
		{{- if .IsAsset -}}
			{{ template "asset" . }}
		{{- else if .ConTableName -}}
			_included_{{ .ConTableName }}.res
		{{- else if .Reference -}}
			{{ template "refColumn" .Reference }}
		{{- else -}}
			{{ .TableName }}.{{ .ColumnName }}
		{{- end -}}
	{{- end -}})
{{- end -}}
{{ range $i, $t := $.Functions }}
CREATE OR REPLACE FUNCTION _{{ .TableName }}_items(locale TEXT, filters _filter[], orderBy TEXT, skip INTEGER, take INTEGER)
RETURNS json AS $$
BEGIN
	WITH filtered AS (
		SELECT _get_sys_ids('{{ .TableName }}', locale, filters, orderBy, skip, take) AS _sys_id
	)
	SELECT json_agg(t) AS res FROM (
		SELECT
			{{ range $i, $c := .Columns }}
				{{- if $i -}},{{- end }}
				{{ if .IsAsset -}}
					{{ template "asset" . }}
				{{- else if .Reference -}}
					{{ template "refColumn" .Reference }}
				{{- else if .ConTableName -}}
					_included_{{ .ConTableName }}.res
				{{- else -}}
					{{ .TableName }}.{{ .ColumnName }}
				{{- end }} AS "{{ .Alias }}"
			{{- end }}
		FROM {{ .TableName }}
		{{ range $i, $c := .Columns }}
			{{ if .IsAsset -}}
				LEFT JOIN {{ .Reference.TableName }} ON {{ .Reference.TableName }}._sys_id = {{ .TableName }}.{{ .ColumnName }} AND {{ .Reference.TableName }}._locale = locale
			{{- /* {{- else if .ConTableName -}}
				{{- $ref:= .Reference -}}
				LEFT JOIN LATERAL (
					SELECT json_agg(l) AS res FROM (
						SELECT
							json_build_object('id', {{ $ref.Reference }}._sys_id) AS sys,
							{{- range $ref.Columns }}
								,
								{{ $ref.Reference }}.{{ $ref.ColumnName }} AS "{{ $ref.Alias }}"
							{{- end }}
						FROM {{ .ConTableName }}
						JOIN {{ $ref.Reference }} on {{ $ref.Reference }}._sys_id = {{ .ConTableName }}.{{ $ref.ForeignKey }} AND {{ $ref.Reference }}._locale = locale
						WHERE {{ .ConTableName }}.{{ $ref.Reference }} = {{ .TableName }}._sys_id AND {{ .ConTableName }}._locale = locale
					) l
				) _included_{{ .ConTableName }} ON true */ -}}
			{{- else if .Reference -}}
				{{- $ref:= .Reference -}}
				LEFT JOIN {{ $ref.TableName }} ON {{ $ref.TableName }}._sys_id = {{ .TableName }}.{{ $ref.ForeignKey }} AND {{ $ref.TableName }}._locale = locale
			{{- end }}
		{{- end }}
		WHERE {{ .TableName }}._locale = locale AND {{ .TableName }}._sys_id IN (SELECT _sys_id FROM filtered)
	) t;
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION _{{ .TableName }}_query(locale TEXT, filters _filter[], orderBy TEXT, skip INTEGER, take INTEGER)
RETURNS _result AS $$
DECLARE
	count integer;
	items json;
	res _result;
BEGIN
	SELECT COUNT(f) FROM (SELECT _get_sys_ids('{{ .TableName }}', locale, filters, '', 0, 0)) AS f INTO count;
	SELECT _{{ .TableName }}_items(locale, filters, orderBy, skip, take) INTO items;
	IF items IS NULL THEN
		items:= '[]'::JSON;
	END IF;
	RETURN ROW(count, items)::_result;
END;
$$ LANGUAGE 'plpgsql';
--
{{- end -}}
`
