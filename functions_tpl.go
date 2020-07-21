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
{{ range $.Tables }}
CREATE OR REPLACE FUNCTION _{{ .TableName }}_items(locale TEXT, filters _filter[], orderBy TEXT, skip INTEGER, take INTEGER)
RETURNS json AS $$
BEGIN
	WITH filtered AS (
		SELECT _get_sys_ids('{{ .TableName }}', locale, filters, orderBy, skip, take) AS _sys_id
	)
	SELECT json_agg(t) AS res FROM (
		SELECT
			{{- range $colidx, $col := .Columns }}
			{{- if $colidx -}},{{- end }}
			{{ if .IsReference -}}_included_{{ .ColumnName }}.res
			{{- else -}}{{ .ColumnName }}
			{{- end }} AS "{{ .Alias }}"
			{{- end }}
			{{ range .Joins -}}
				json_build_object('sys', 
					json_build_object('id', {{ .Reference }}.{{ .ForeignKey }})
					{{- range $colidx, $col := .Columns }}
						,
						'{{ .Alias }}', {{ .Reference }}.{{ .ColumnName }}
					{{- end }}) as {{ .Alias }}
			{{- end }}
			{{ range .LateralJoins -}}
				,
				_included_{{ .ColumnName }}.res.res AS "{{ .ColumnName }}"
			{{- end }}
		FROM {{ .TableName }}
		{{ range .Joins -}}
			LEFT JOIN {{ .Reference }} ON {{ .Reference }}.{{ .ForeignKey }} = {{ .TableName }}.{{ .Reference }} AND {{ .Reference }}._locale = locale
		{{- end }}
		{{ range .LateralJoins -}}
			LEFT JOIN LATERAL (
				SELECT json_agg(l) AS res FROM (
					SELECT
						json_build_object('id', {{ .Reference }}.{{ .ForeignKey }}) AS sys,
						{{- range $colidx, $col := .Columns }}
							,
							{{ .Reference }}.{{ .ColumnName }} AS "{{ .Alias }}"
						{{- end }}
					FROM {{ .ConTableName }}
					JOIN {{ .Reference }} on {{ .Reference }}.{{ .ForeignKey }} = {{ .ConTableName }}.{{ .Reference }} AND {{ .Reference }}._locale = locale
					WHERE {{ .ConTableName }}.{{ .TableName }} = {{ .TableName }}.{{ .ForeignKey }} AND {{ .ConTableName }}._locale = locale
				) l
			) _included_{{ .ColumnName }} ON true
		{{- end }}
		{{ range .References -}}
			LEFT JOIN {{ .Reference }} ON {{ .Reference }}.{{ .TableName }} = {{ .TableName }}.{{ .ForeignKey }} AND {{ .Reference }}._locale = locale
			LEFT JOIN LATERAL (SELECT )
		{{- end }}
		WHERE {{ .TableName }}._locale = locale AND {{ .TableName }}._sys_id IN (SELECT _sys_id FROM filtered)
	) t
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
{{- end -}}
`
