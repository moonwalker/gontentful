package gontentful

const pgRefreshMatViewsTemplate = `
{{ range $i, $l := $.Locales }}
REFRESH MATERIALIZED VIEW "mv_{{ $.TableName }}_{{ .Code | ToLower }}";
{{- end }}`

const pgRefreshMatViewsGetDepsTemplate = `
WITH RECURSIVE refs AS (
	SELECT '{{ . }}' AS "tablename", 1 AS "rl" 
	UNION ALL
	SELECT tr.tablename, r.rl + 1 FROM refs AS r
	JOIN table_references tr ON tr.reference = r.tablename
	WHERE r.rl < 3
)
SELECT DISTINCT refs.tablename FROM refs;
`

const pgFuncTemplate = `
{{- if not $.ContentTypePublish }}
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
	IF NOT EXISTS (SELECT 1 FROM pg_type AS pt JOIN pg_namespace as pn ON (pt.typnamespace = pn.oid) WHERE pn.nspname = '{{ $.SchemaName }}' and pt.typname = '_result') THEN
		CREATE TYPE _result AS (
			count INTEGER,
			items JSON
		);
	END IF;
END $$;
--
{{ end -}}
{{ range $i, $t := $.Tables }}
{{- if $.DropTables }}
DROP FUNCTION IF EXISTS {{ .TableName }}_view CASCADE;
{{ end -}}
{{ end }}
--
{{- define "assetRef" -}}
(CASE WHEN 
	{{ .Reference.JoinAlias }}._sys_id IS NULL THEN NULL 
ELSE
json_build_object(
	'title', {{ .Reference.JoinAlias }}.title,
	'description', {{ .Reference.JoinAlias }}.description,
	'file', json_build_object(
		'contentType', {{ .Reference.JoinAlias }}.content_type,
		'fileName', {{ .Reference.JoinAlias }}.file_name,
		'url', {{ .Reference.JoinAlias }}.url)
)
END)
{{- end -}}
{{- define "assetCon" -}}
		json_build_object(
			'id', {{ .Reference.JoinAlias }}._sys_id,
			'createdAt', {{ .Reference.JoinAlias }}._created_at,
			'updatedAt', {{ .Reference.JoinAlias }}._updated_at
		) AS sys,		
		{{ .Reference.JoinAlias }}.title AS "title",
		{{ .Reference.JoinAlias }}.description AS "description",
		json_build_object(
			'contentType', {{ .Reference.JoinAlias }}.content_type,
			'fileName', {{ .Reference.JoinAlias }}.file_name,
			'url', {{ .Reference.JoinAlias }}.url
		) AS "file"
{{- end -}}
{{- define "refColumn" -}} 
{{ if .Localized -}}
	(CASE WHEN {{ .JoinAlias }}._sys_id IS NULL THEN NULL ELSE json_build_object(
		'sys', json_build_object(
			'id', {{ .JoinAlias }}._sys_id,
			'createdAt', {{ .JoinAlias }}._created_at,
			'updatedAt', {{ .JoinAlias }}._updated_at
		)
{{- else -}}
	(CASE WHEN {{ .JoinAlias }}._sys_id IS NULL THEN NULL ELSE json_build_object(
		'sys', json_build_object(
			'id', {{ .JoinAlias }}._sys_id,
			'createdAt', {{ .JoinAlias }}._created_at,
			'updatedAt', {{ .JoinAlias }}._updated_at
		)
{{- end -}}
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
	json_build_object(
		'id', {{ .JoinAlias }}._sys_id,
		'createdAt', {{ .JoinAlias }}._created_at,
		'updatedAt', {{ .JoinAlias }}._updated_at
	) AS sys
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
				LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }} ON {{ .Reference.JoinAlias }}._id = {{ .ConTableName }}.{{ .Reference.TableName }}
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
--
{{- define "query" -}}
CREATE OR REPLACE FUNCTION {{ .TableName }}_query(localeArg TEXT, filters TEXT[], orderBy TEXT, skip INTEGER, take INTEGER)
RETURNS _result AS $body$
DECLARE 
	res _result;
	qs text := '';
	filter text;
	counter integer := 0; 
BEGIN
	qs:= 'WITH filtered AS (
		SELECT COUNT(*) OVER() AS _count, row_number() OVER(';

	IF orderBy <> '' THEN
		qs:= qs || ' ORDER BY ' || orderBy;
	END IF;

	qs:= qs || ') AS _idx,' || '{{ .TableName }}.* FROM "mv_{{ .TableName}}_' || lower(localeArg) || '" {{ .TableName }}';
	
	IF filters IS NOT NULL THEN
		qs := qs || ' WHERE';
		FOREACH filter IN ARRAY filters LOOP
			if counter > 0 then
				qs := qs || ' AND ';
	 		end if;
			qs := qs || ' (' || '{{ .TableName }}' || '.' || filter || ')';
			counter := counter + 1;
		END LOOP;
	END IF;

	IF skip <> 0 THEN
	qs:= qs || ' OFFSET ' || skip;
	END IF;

	IF take <> 0 THEN
	qs:= qs || ' LIMIT ' || take;
	END IF;

	qs:= qs || ') ';
			
	qs:= qs || 'SELECT (SELECT _count FROM filtered LIMIT 1)::INTEGER, json_agg(t)::json FROM (
	SELECT json_build_object(
		''id'', {{ .TableName }}._sys_id,
		''createdAt'', {{ .TableName }}._created_at,
		''updatedAt'', {{ .TableName }}._updated_at
	) AS sys
	{{- range .Columns -}}
		,
		{{ .TableName }}.{{ .ColumnName }} AS "{{ .Alias }}"
	{{- end }}
	FROM filtered {{ .TableName }}';

	qs:= qs || ' ORDER BY {{ .TableName }}._idx ) t;';

	EXECUTE qs INTO res;

	IF res.items IS NULL THEN
		res.items:= '[]'::JSON;
		res.count:=0::INTEGER;
	END IF;
	RETURN res;
END $body$ LANGUAGE 'plpgsql';
{{- end -}}
--
{{ range $i, $t := $.Functions }}
{{ if $.ContentSchema -}}
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_tables WHERE  schemaname = '{{ $.ContentSchema }}' AND tablename  = 'game_{{ .TableName}}') THEN
		CREATE OR REPLACE FUNCTION {{ .TableName }}_query(localeArg TEXT, filters TEXT[], orderBy TEXT, skip INTEGER, take INTEGER)
		RETURNS _result AS $body$
		DECLARE 
			res _result;
			qs text := '';
			filter text;
			counter integer := 0; 
		BEGIN
			qs:= 'WITH filtered AS (
				SELECT COUNT(*) OVER() AS _count, row_number() OVER(';
		
			IF orderBy <> '' THEN
				qs:= qs || ' ORDER BY ' || orderBy;
			END IF;
		
			qs:= qs || ') AS _idx,' || '{{ .TableName }}.* FROM "mv_{{ .TableName}}_' || lower(localeArg) || '" {{ .TableName }}';
			
			IF filters IS NOT NULL THEN
				qs := qs || ' WHERE';
				FOREACH filter IN ARRAY filters LOOP
					if counter > 0 then
						qs := qs || ' AND ';
					end if;
					qs := qs || ' (' || '{{ .TableName }}' || '.' || filter || ')';
					counter := counter + 1;
				END LOOP;
			END IF;
		
			IF skip <> 0 THEN
			qs:= qs || ' OFFSET ' || skip;
			END IF;
		
			IF take <> 0 THEN
			qs:= qs || ' LIMIT ' || take;
			END IF;
		
			qs:= qs || ') ';
					
			qs:= qs || 'SELECT (SELECT _count FROM filtered LIMIT 1)::INTEGER, json_agg(t)::json FROM (
			SELECT json_build_object(
				''id'', {{ .TableName }}._sys_id,
				''createdAt'', {{ .TableName }}._created_at,
				''updatedAt'', {{ .TableName }}._updated_at
			) AS sys
			{{- range .Columns -}}
				,
				{{ if and ($.ContentSchema) (.ColumnName | Overwritable) -}}
				COALESCE(c_{{ .TableName }}.{{ .ColumnName }}, {{ .TableName }}.{{ .ColumnName }}) AS "{{ .Alias }}"
				{{- else -}}
				{{ .TableName }}.{{ .ColumnName }} AS "{{ .Alias }}"
				{{- end -}}
				{{- end }}
			FROM filtered {{ .TableName }}
			{{ if $.ContentSchema -}}
			LEFT JOIN {{ $.ContentSchema }}."mv_game_{{ .TableName}}_' || lower(localeArg) || '" c_{{ .TableName }} ON (c_{{ .TableName }}.slug = {{ .TableName }}.slug)';
			{{- else -}}
			';
			{{- end }}											

			qs:= qs || ' ORDER BY {{ .TableName }}._idx ) t;';

			EXECUTE qs INTO res;

			IF res.items IS NULL THEN
				res.items:= '[]'::JSON;
				res.count:=0::INTEGER;
			END IF;
			RETURN res;
		END $body$ LANGUAGE 'plpgsql';
	END IF;
END $$;
DO $$
BEGIN
	IF NOT EXISTS (SELECT FROM pg_tables WHERE  schemaname = '{{ $.ContentSchema }}' AND tablename  = 'game_{{ .TableName}}') THEN
		{{ template "query" . }}	
	END IF;
END $$;
{{- else -}}
{{ template "query" . }}	
{{-  end -}}
--
CREATE OR REPLACE FUNCTION {{ .TableName }}_view(localeArg TEXT)
RETURNS table(_id text, _sys_id text {{- range .Columns -}}
		,
		{{ if eq .ColumnName "limit" -}}_{{- end -}}
		{{- .ColumnName }} {{ .SqlType -}} 
	{{- end -}}
	, _created_at timestamp
	, _updated_at timestamp) AS $$
BEGIN
	RETURN QUERY
		SELECT
			{{ .TableName }}._id AS _id,
			{{ .TableName }}._sys_id AS _sys_id
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
			{{- end }} AS "{{ .ColumnName }}"
		{{- end }},
			{{ .TableName }}._created_at AS _created_at,
			{{ .TableName }}._updated_at AS _updated_at
		FROM {{ .TableName }}
		{{- range .Columns -}}
			{{ template "join" . }}
		{{- end }}
		WHERE {{ .TableName }}._locale = localeArg;
END;
$$ LANGUAGE 'plpgsql';
--
{{ range $i, $l := $.Locales }}

{{ if $.DropTables -}}
CREATE MATERIALIZED VIEW IF NOT EXISTS "mv_{{ $t.TableName }}_{{ .Code | ToLower }}" AS SELECT * FROM {{ $t.TableName }}_view('{{ .Code | ToLower }}');
{{ else -}}
CREATE MATERIALIZED VIEW IF NOT EXISTS "mv_{{ $t.TableName }}_{{ .Code | ToLower }}" AS SELECT * FROM {{ $t.TableName }}_view('{{ .Code | ToLower }}') WITH NO DATA;
{{- end }}
CREATE UNIQUE INDEX IF NOT EXISTS "mv_{{ $t.TableName }}_{{ .Code | ToLower }}_idx" ON "mv_{{ $t.TableName }}_{{ .Code | ToLower }}" (_id);
--
{{ range $cfi, $cfl := .CFLocales }}
CREATE OR REPLACE VIEW "mv_{{ $t.TableName }}_{{ $cfl | ToLower }}" AS SELECT * FROM "mv_{{ $t.TableName }}_{{ $l.Code | ToLower }}";
{{- end }}
--
{{- end }}
{{- end }}
--
{{- range $i, $t := $.DeleteTriggers }}
CREATE OR REPLACE FUNCTION {{ .TableName }}_delete_trigger() 
   RETURNS TRIGGER 
AS $$
BEGIN
	{{- range $idx, $c := .ConTables }}
	DELETE FROM {{ . }} where {{ . }}.{{ $t.TableName }} = OLD._id;
	{{- end }}
	RETURN OLD;
END
$$ LANGUAGE 'plpgsql';
--
DROP TRIGGER IF EXISTS {{ .TableName }}_delete
ON {{ .TableName }};

CREATE TRIGGER {{ .TableName }}_delete 
AFTER DELETE 
ON {{ .TableName }} 
FOR EACH ROW 
EXECUTE PROCEDURE {{ .TableName }}_delete_trigger();

{{- end }}
`

const pgFuncPublishTemplate = `
{{ range $i, $t := $.Functions }}
	DROP VIEW IF EXISTS "mv_{{ $t.TableName }}_{{ $.Locale | ToLower }}"; 
	CREATE OR REPLACE VIEW "mv_{{ $t.TableName }}_{{ $.Locale | ToLower }}" AS SELECT * FROM "mv_{{ $t.TableName }}_{{ $.FallbackLocale | ToLower }}";
{{- end }}
`
