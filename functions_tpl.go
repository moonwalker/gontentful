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
{{ range $i, $t := $.Tables }}
{{- if $.DropTables }}
DROP FUNCTION IF EXISTS {{ .TableName }}_view CASCADE;
{{ end -}}
{{ end }}
--
{{- define "assetRef" -}}
(CASE WHEN 
	{{ .Reference.JoinAlias }}._sys_id IS NULL AND 
	{{ .Reference.JoinAlias }}_fallbacklocale._sys_id IS NULL AND 
	{{ .Reference.JoinAlias }}_deflocale._sys_id IS NULL THEN NULL 
ELSE
json_build_object(
	'title', COALESCE({{ .Reference.JoinAlias }}.title, {{ .Reference.JoinAlias }}_fallbacklocale.title, {{ .Reference.JoinAlias }}_deflocale.title),
	'description', COALESCE({{ .Reference.JoinAlias }}.description, {{ .Reference.JoinAlias }}_fallbacklocale.description, {{ .Reference.JoinAlias }}_deflocale.description),
	'file', (CASE WHEN {{ .Reference.JoinAlias }}._sys_id IS NULL THEN 
		(CASE WHEN {{ .Reference.JoinAlias }}_fallbacklocale._sys_id IS NULL THEN
			json_build_object(
				'contentType', {{ .Reference.JoinAlias }}_deflocale.content_type,
				'fileName', {{ .Reference.JoinAlias }}_deflocale.file_name,
				'url', {{ .Reference.JoinAlias }}_deflocale.url
			)
		ELSE 
		json_build_object(
			'contentType', {{ .Reference.JoinAlias }}_fallbacklocale.content_type,
			'fileName', {{ .Reference.JoinAlias }}_fallbacklocale.file_name,
			'url', {{ .Reference.JoinAlias }}_fallbacklocale.url
		) END)	
	ELSE 
		json_build_object(
			'contentType', {{ .Reference.JoinAlias }}.content_type,
			'fileName', {{ .Reference.JoinAlias }}.file_name,
			'url', {{ .Reference.JoinAlias }}.url
		)
	END)
)
END)
{{- end -}}
{{- define "assetCon" -}}
		json_build_object('id', COALESCE({{ .Reference.JoinAlias }}._sys_id, {{ .Reference.JoinAlias }}_fallbacklocale._sys_id, {{ .Reference.JoinAlias }}_deflocale._sys_id)) AS sys,
								(CASE WHEN {{ .Reference.JoinAlias }}._sys_id IS NULL THEN (CASE WHEN {{ .Reference.JoinAlias }}_fallbacklocale._sys_id IS NULL THEN {{ .Reference.JoinAlias }}_deflocale.title ELSE {{ .Reference.JoinAlias }}_fallbacklocale.title END) ELSE {{ .Reference.JoinAlias }}.title END) AS "title",
								(CASE WHEN {{ .Reference.JoinAlias }}._sys_id IS NULL THEN (CASE WHEN {{ .Reference.JoinAlias }}_fallbacklocale._sys_id IS NULL THEN {{ .Reference.JoinAlias }}_deflocale.description ELSE {{ .Reference.JoinAlias }}_fallbacklocale.description END) ELSE {{ .Reference.JoinAlias }}.description END) AS "description",
								(CASE WHEN {{ .Reference.JoinAlias }}._sys_id IS NULL THEN 
									(CASE WHEN {{ .Reference.JoinAlias }}_fallbacklocale._sys_id IS NULL THEN
										json_build_object(
											'contentType', {{ .Reference.JoinAlias }}_deflocale.content_type,
											'fileName', {{ .Reference.JoinAlias }}_deflocale.file_name,
											'url', {{ .Reference.JoinAlias }}_deflocale.url
										)
									ELSE 
									json_build_object(
										'contentType', {{ .Reference.JoinAlias }}_fallbacklocale.content_type,
										'fileName', {{ .Reference.JoinAlias }}_fallbacklocale.file_name,
										'url', {{ .Reference.JoinAlias }}_fallbacklocale.url
									) END)	
								ELSE 
									json_build_object(
										'contentType', {{ .Reference.JoinAlias }}.content_type,
										'fileName', {{ .Reference.JoinAlias }}.file_name,
										'url', {{ .Reference.JoinAlias }}.url
									)
								END) AS "file"
{{- end -}}
{{- define "refColumn" -}} 
{{ if .Localized -}}
(CASE WHEN COALESCE({{ .JoinAlias }}._sys_id, {{ .JoinAlias }}_fallbacklocale._sys_id, {{ .JoinAlias }}_deflocale._sys_id) IS NULL THEN NULL ELSE json_build_object(
	'sys', json_build_object('id', COALESCE({{ .JoinAlias }}._sys_id, {{ .JoinAlias }}_fallbacklocale._sys_id, {{ .JoinAlias }}_deflocale._sys_id))
{{- else -}}
(CASE WHEN {{ .JoinAlias }}._sys_id IS NULL THEN NULL ELSE json_build_object(
	'sys', json_build_object('id', {{ .JoinAlias }}._sys_id)
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
						{{ if .Localized -}}
							COALESCE({{ .JoinAlias }}.{{ .ColumnName }}, {{ .JoinAlias }}_fallbacklocale.{{ .ColumnName }}, {{ .JoinAlias }}_deflocale.{{ .ColumnName }}) 
						{{- else -}}
							{{ .JoinAlias }}.{{ .ColumnName }}
						{{- end -}}	
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
							{{ if .Localized -}}
								COALESCE({{ .JoinAlias }}.{{ .ColumnName }}, {{ .JoinAlias }}_fallbacklocale.{{ .ColumnName }}, {{ .JoinAlias }}_deflocale.{{ .ColumnName }}) 
							{{- else -}}
								{{ .JoinAlias }}.{{ .ColumnName }}
							{{- end -}}	
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
				{{ if .Localized -}}
				LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }} ON {{ .Reference.JoinAlias }}._sys_id = {{ .ConTableName }}.{{ .Reference.TableName }}_sys_id AND {{ .Reference.JoinAlias }}._locale = localeArg
				{{- else -}}
				LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }} ON {{ .Reference.JoinAlias }}._id = {{ .ConTableName }}.{{ .Reference.TableName }}
				{{- end }}
				{{ if or .Localized .IsAsset .Reference.HasLocalized -}}
				-- Join (Localized:{{ .Localized }}, IsAsset:{{ .IsAsset }}, Reference HasLocalized:{{ .Reference.HasLocalized }}, Reference Localized:{{ .Reference.Localized }})
				LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }}_fallbacklocale ON {{ .Reference.JoinAlias }}_fallbacklocale._sys_id = {{ .ConTableName }}.{{ .Reference.TableName }}_sys_id AND {{ .Reference.JoinAlias }}_fallbacklocale._locale = fallbackLocaleArg
				LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }}_deflocale ON {{ .Reference.JoinAlias }}_deflocale._sys_id = {{ .ConTableName }}.{{ .Reference.TableName }}_sys_id AND {{ .Reference.JoinAlias }}_deflocale._locale = defLocaleArg
				{{- end -}}
				{{- range .Reference.Columns }}
				{{- template "join" . }}
				{{- end }}
				WHERE {{ .ConTableName }}.{{ .TableName }} = 
				{{- if .Localized -}}	
				-- IsLocalized join
				(CASE WHEN {{ .JoinAlias }}.{{ .ColumnName }} IS NULL THEN (CASE WHEN {{ .JoinAlias }}_fallbacklocale.{{ .ColumnName }} IS NULL THEN {{ .JoinAlias }}_deflocale._id ELSE {{ .JoinAlias }}_fallbacklocale._id END) ELSE {{ .JoinAlias }}._id END)
				{{- else -}}
				{{ .JoinAlias }}._id
				{{- end }}
				ORDER BY {{ .ConTableName }}._id
			) l
		) _included_{{ .Reference.JoinAlias }} ON true
	{{- else if .Reference }}
		{{ if .Localized -}}
		LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }} ON {{ .Reference.JoinAlias }}._sys_id = COALESCE({{ .JoinAlias }}.{{ .Reference.ForeignKey }},{{ .JoinAlias }}_fallbacklocale.{{ .Reference.ForeignKey }} ,{{ .JoinAlias }}_deflocale.{{ .Reference.ForeignKey }}) AND {{ .Reference.JoinAlias }}._locale = localeArg
		{{- else -}}
		LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }} ON {{ .Reference.JoinAlias }}._sys_id = {{ .JoinAlias }}.{{ .Reference.ForeignKey }} AND {{ .Reference.JoinAlias }}._locale = localeArg
		{{- end -}}
		{{ if or .Localized .IsAsset .Reference.HasLocalized }}
		{{ if .IsAsset -}}
		-- IsAsset join
		LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }}_fallbacklocale ON {{ .Reference.JoinAlias }}_fallbacklocale._sys_id = {{ .JoinAlias }}.{{ .Reference.ForeignKey }} AND {{ .Reference.JoinAlias }}_fallbacklocale._locale = fallbackLocaleArg
		LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }}_deflocale ON {{ .Reference.JoinAlias }}_deflocale._sys_id = {{ .JoinAlias }}.{{ .Reference.ForeignKey }} AND {{ .Reference.JoinAlias }}_deflocale._locale = defLocaleArg
		{{- else -}}
		-- Reference (Localized:{{ .Localized }}, Reference Localized:{{ .Reference.Localized }}, Reference HasLocalized:{{ .Reference.HasLocalized }})
		{{ if .Reference.Localized }}
		LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }}_fallbacklocale ON {{ .Reference.JoinAlias }}_fallbacklocale._sys_id = COALESCE({{ .JoinAlias }}.{{ .Reference.ForeignKey }},{{ .JoinAlias }}_fallbacklocale.{{ .Reference.ForeignKey }} ,{{ .JoinAlias }}_deflocale.{{ .Reference.ForeignKey }}) AND {{ .Reference.JoinAlias }}_fallbacklocale._locale = fallbackLocaleArg
		LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }}_deflocale ON {{ .Reference.JoinAlias }}_deflocale._sys_id = COALESCE({{ .JoinAlias }}.{{ .Reference.ForeignKey }},{{ .JoinAlias }}_fallbacklocale.{{ .Reference.ForeignKey }} ,{{ .JoinAlias }}_deflocale.{{ .Reference.ForeignKey }}) AND {{ .Reference.JoinAlias }}_deflocale._locale = defLocaleArg
		{{- else -}}
		LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }}_fallbacklocale ON {{ .Reference.JoinAlias }}_fallbacklocale._sys_id = {{ .JoinAlias }}.{{ .Reference.ForeignKey }} AND {{ .Reference.JoinAlias }}_fallbacklocale._locale = fallbackLocaleArg
		LEFT JOIN {{ .Reference.TableName }} {{ .Reference.JoinAlias }}_deflocale ON {{ .Reference.JoinAlias }}_deflocale._sys_id = {{ .JoinAlias }}.{{ .Reference.ForeignKey }} AND {{ .Reference.JoinAlias }}_deflocale._locale = defLocaleArg
		{{- end -}}
		{{- end -}}
		{{- end -}}
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
	SELECT json_build_object(''id'', {{ .TableName }}._sys_id) AS sys
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
			SELECT json_build_object(''id'', {{ .TableName }}._sys_id) AS sys
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
CREATE OR REPLACE FUNCTION {{ .TableName }}_view(localeArg TEXT, fallbackLocaleArg TEXT, defLocaleArg TEXT)
RETURNS table(_id text, _sys_id text {{- range .Columns -}}
		,
		{{ if eq .ColumnName "limit" -}}_{{- end -}}
		{{- .ColumnName }} {{ .SqlType -}} 
	{{- end -}}
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
			{{ if .Localized -}}
				COALESCE({{ .TableName }}.{{ .ColumnName }}, {{ .TableName }}_fallbacklocale.{{ .ColumnName }}, {{ .TableName }}_deflocale.{{ .ColumnName }}) 
			{{- else -}}
				{{ .TableName }}.{{ .ColumnName }}
			{{- end -}}
			{{- end }} AS "{{ .ColumnName }}"
		{{- end }},
			{{ .TableName }}._updated_at AS _updated_at
		FROM {{ .TableName }}
		{{ if .HasLocalized -}}
		LEFT JOIN {{ .TableName }} {{ .TableName }}_fallbacklocale ON {{ .TableName }}._sys_id = {{ .TableName }}_fallbacklocale._sys_id AND {{ .TableName }}_fallbacklocale._locale = fallbackLocaleArg
		LEFT JOIN {{ .TableName }} {{ .TableName }}_deflocale ON {{ .TableName }}._sys_id = {{ .TableName }}_deflocale._sys_id AND {{ .TableName }}_deflocale._locale = defLocaleArg
		{{- end }}
		{{- range .Columns -}}
			{{ template "join" . }}
		{{- end }}
		WHERE {{ .TableName }}._locale = localeArg;
END;
$$ LANGUAGE 'plpgsql';
--
{{ range $i, $l := $.Locales }}
{{ $fallbackLocale := .FallbackCode }}
{{- if eq $fallbackLocale "" -}}
	{{ $fallbackLocale := "en" }}
{{- end -}}

{{- if $.DropTables -}}
CREATE MATERIALIZED VIEW IF NOT EXISTS "mv_{{ $t.TableName }}_{{ .Code | ToLower }}" AS SELECT * FROM {{ $t.TableName }}_view('{{ .Code | ToLower }}', '{{ $fallbackLocale | ToLower }}', 'en');
{{- else -}}
CREATE MATERIALIZED VIEW IF NOT EXISTS "mv_{{ $t.TableName }}_{{ .Code | ToLower }}" AS SELECT * FROM {{ $t.TableName }}_view('{{ .Code | ToLower }}', '{{ $fallbackLocale | ToLower }}', 'en') WITH NO DATA;
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
