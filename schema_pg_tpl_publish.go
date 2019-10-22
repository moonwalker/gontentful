package gontentful

const pgTemplatePublish = `BEGIN;
{{ if .Drop }}
DROP SCHEMA IF EXISTS {{ $.SchemaName }} CASCADE;
{{ end -}}
CREATE SCHEMA IF NOT EXISTS {{ $.SchemaName }};
--
DROP TYPE IF EXISTS {{ $.SchemaName }}._meta CASCADE;
CREATE TYPE {{ $.SchemaName }}._meta AS (
	name TEXT,
	type TEXT,
	items_type TEXT,
	link_type TEXT,
	is_localized BOOLEAN
);
DROP TYPE IF EXISTS {{ $.SchemaName }}._filter CASCADE;
CREATE TYPE {{ $.SchemaName }}._filter AS (
	field TEXT,
	comparer TEXT,
	values TEXT[]
);
DROP TYPE IF EXISTS {{ $.SchemaName }}._result CASCADE;
CREATE TYPE {{ $.SchemaName }}._result AS (
	count INTEGER,
	items JSON
);
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._get_meta(tableName text)
RETURNS SETOF {{ $.SchemaName }}._meta AS $$
BEGIN
	 RETURN QUERY EXECUTE 'SELECT
		name,
		type,
		items_type,
		link_type,
		is_localized
        FROM {{ $.SchemaName }}.' || tableName || '$meta';

END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._fmt_column_name(colum text)
RETURNS text AS $$
DECLARE
	splits text[];
BEGIN
	splits:= string_to_array(colum, '_');
	RETURN splits[1] || replace(INITCAP(array_to_string(splits[2:], ' ')), ' ', '');
END;
$$ LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._fmt_value(val text, isText boolean, isWildcard boolean, isList boolean)
RETURNS text AS $$
DECLARE
	res text;
	v text;
	isFirst boolean:= true;
BEGIN
	IF isText THEN
		IF isWildcard THEN
			RETURN '''%' || val || '%''';
		ELSEIF isList THEN
			FOREACH v IN ARRAY string_to_array(val, ',') LOOP
				IF isFirst THEN
					isFirst:= false;
					res:= '';
				ELSE
					res:= res || ',';
				END IF;
				res:= res || '''' || v || '''';
			END LOOP;
			RETURN res;
		END IF;
		RETURN '''' || val || '''';
	END IF;
	RETURN val;
END;
$$ LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._fmt_comparer(comparer text, fmtVal text, isArray boolean)
RETURNS text AS $$
BEGIN
	IF fmtVal IS NOT NULL THEN
		IF comparer = '' THEN
			RETURN ' = ' || fmtVal;
		ELSEIF  comparer = 'ne' THEN
			RETURN ' <> ' || fmtVal;
		ELSEIF  comparer = 'exists' THEN
			RETURN ' <> NULL';
		ELSEIF  comparer = 'lt' THEN
			RETURN ' < ' || fmtVal;
		ELSEIF  comparer = 'lte' THEN
			RETURN ' <= ' || fmtVal;
		ELSEIF  comparer = 'gt' THEN
			RETURN ' > ' || fmtVal;
		ELSEIF  comparer = 'gte' THEN
			RETURN ' >= ' || fmtVal;
		ELSEIF comparer = 'match' THEN
			RETURN ' LIKE ' || fmtVal;
		ELSEIF comparer = 'in' THEN
			IF isArray THEN
				RETURN 	' && ARRAY[' || fmtVal || ']';
			END IF;
			RETURN 	' = ANY(ARRAY[' || fmtVal || '])';
		ELSEIF comparer = 'nin' THEN
			IF isArray THEN
				RETURN 	' && ARRAY[' || fmtVal || '] = false';
			END IF;
			RETURN 	' <> ANY(ARRAY[' || fmtVal || '])';
		END IF;
	END IF;
	RETURN '';
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._fmt_clause(meta {{ $.SchemaName }}._meta, tableName text, defaultLocale text, locale text, comparer text, filterValues text[], field text, subField text)
RETURNS text AS $$
DECLARE
	colType text;
	isArray boolean;
	isText boolean;
	isWildcard boolean;
	isList boolean;
	fmtVal text:= '';
	isFirst boolean:= true;
	val text;
	fmtComp text;
BEGIN
	IF meta IS NULL THEN -- sys_id
		colType:= 'Link';
	ELSEIF meta.items_type <> '' THEN
		colType:= meta.items_type;
		isArray:= true;
	ELSE
		colType:= meta.type;
	END IF;

	IF colType ='Symbol' OR colType='Text' OR colType ='Date' OR colType ='Link' THEN
		isText:= true;
	END IF;

	IF isText AND comparer = 'match' THEN
		isWildcard:= true;
	END IF;

	IF isText AND (comparer = 'in' OR comparer = 'nin') THEN
		isList:= true;
	END IF;

	IF isArray OR isList THEN
		FOREACH val IN ARRAY filterValues LOOP
			IF isFirst THEN
		    	isFirst := false;
		    ELSE
		    	fmtVal := fmtVal || ',';
		    END IF;
			fmtVal:= fmtVal || {{ $.SchemaName }}._fmt_value(val, isText, isWildcard, isList);
		END LOOP;
		IF subField IS NOT NULL THEN
			RETURN 'EXISTS (SELECT FROM json_array_elements(_included_' || meta.name || '.res) js WHERE js ->> ''' || subField || '''' || {{ $.SchemaName }}._fmt_comparer(comparer, fmtVal, false) || ')';
		END IF;
		IF meta.is_localized AND locale <> defaultLocale THEN
			RETURN 'COALESCE(' || tableName || '$' || locale || '.' || meta.name || ',' ||
			tableName || '$' || defaultLocale || '.' || meta.name || ')' || {{ $.SchemaName }}._fmt_comparer(comparer, fmtVal, isArray);
		END IF;
		RETURN tableName || '$' || defaultLocale || '.' || meta.name || {{ $.SchemaName }}._fmt_comparer(comparer, fmtVal, isArray);
	END IF;

	FOREACH val IN ARRAY filterValues LOOP
		fmtComp:= {{ $.SchemaName }}._fmt_comparer(comparer, {{ $.SchemaName }}._fmt_value(val, isText, isWildcard, isList), false);
		IF fmtComp <> '' THEN
			IF fmtVal <> '' THEN
	    		fmtVal := fmtVal || ' OR ';
			END IF;
			IF meta IS NOT NULL THEN
				IF subField IS NOT NULL THEN
					fmtVal := fmtVal || '(_included_' || field || '.res ->> ''' || subField || ''')::text' || fmtComp;
				ELSEIF meta.is_localized AND locale <> defaultLocale THEN
					fmtVal := fmtVal || 'COALESCE(' || tableName || '$' || locale || '.' || meta.name || ',' ||
					tableName || '$' || defaultLocale || '.' || meta.name || ')' || fmtComp;
				ELSE
					fmtVal := fmtVal || tableName || '$' || defaultLocale || '.' || meta.name || fmtComp;
				END IF;
			ELSE
				fmtVal := fmtVal || tableName || '$' || defaultLocale || '.' || field || fmtComp;
			END IF;
	    END IF;
	END LOOP;
	RETURN fmtVal;
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._build_critertia(tableName text, meta {{ $.SchemaName }}._meta, defaultLocale text, locale text)
RETURNS text AS $$
DECLARE
	c text;
	f text;
BEGIN
	c:= meta.link_type || '$' || defaultLocale || '.sys_id = ';
	IF meta.is_localized AND locale <> defaultLocale THEN
		f := 'COALESCE(' || tableName || '$' || locale || '.' || meta.name || ',' ||
		tableName || '$' || defaultLocale || '.' || meta.name || ')';
	ELSE
		f := tableName || '$' || defaultLocale || '.' || meta.name;
	END IF;

	IF meta.items_type <> '' THEN
		f := 'ANY(' || f || ')';
	END IF;

	RETURN c || f;
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._include_join(tableName TEXT, criteria TEXT, isArray BOOLEAN, locale TEXT, defaultLocale TEXT, suffix TEXT, includeDepth INTEGER)
RETURNS text AS $$
DECLARE
	qs text;
	hasLocalized boolean := false;
	joinedTables {{ $.SchemaName }}._meta[];
	meta {{ $.SchemaName }}._meta;
	crit text;
BEGIN
	qs := 'json_build_object(';

	-- qs:= qs || tableName || '$' || defaultLocale || '.sys_id as sys_id, ';
	qs:= qs || '''sys'',json_build_object(''id'','  || tableName || '$' || defaultLocale || '.sys_id)';

	IF tableName = '_asset' THEN
		qs := qs || ',';

		IF locale <> defaultLocale THEN
			hasLocalized:= true;
		END IF;

		IF hasLocalized THEN
			qs := qs ||
			'''title'',' || 'COALESCE(' || tableName || '$' || locale || '.title,' || tableName || '$' || defaultLocale || '.title),' ||
			'''description'',' || 'COALESCE(' || tableName || '$' || locale || '.description,' || tableName || '$' || defaultLocale || '.description),' ||
			'''file'',json_build_object(' ||
				'''contentType'',COALESCE(' || tableName || '$' || locale || '.content_type,' || tableName || '$' || defaultLocale || '.content_type),' ||
				'''fileName'',COALESCE(' || tableName || '$' || locale || '.file_name,' || tableName || '$' || defaultLocale || '.file_name),' ||
				'''url'',COALESCE(' || tableName || '$' || locale || '.url,' || tableName || '$' || defaultLocale || '.url))';
		ELSE
			qs := qs ||
			'''title'',' || tableName || '$' || defaultLocale || '.title,' ||
			'''description'',' || tableName || '$' || defaultLocale || '.description,' ||
			'''file'',json_build_object(' ||
				'''contentType'',' || tableName || '$' || defaultLocale || '.content_type,' ||
				'''fileName'',' || tableName || '$' || defaultLocale || '.file_name,' ||
				'''url'',' || tableName || '$' || defaultLocale || '.url)';
		END IF;
	ELSE

		FOR meta IN SELECT * FROM {{ $.SchemaName }}._get_meta(tableName) LOOP
			qs := qs || ', ';

			qs := qs || '''' || {{ $.SchemaName }}._fmt_column_name(meta.name) || ''',';

			IF meta.is_localized AND locale <> defaultLocale THEN
				hasLocalized:= true;
			END IF;

			IF meta.link_type <> '' AND includeDepth > 0 THEN
				qs := qs || '_included_' || meta.name || '.res';
				joinedTables:= joinedTables || meta;
			ELSEIF hasLocalized THEN
				qs := qs || 'COALESCE(' || tableName || '$' || locale || '.' || meta.name || ',' ||
					tableName || '$' || defaultLocale || '.' || meta.name || ')';
			ELSE
			   	qs := qs || tableName || '$' || defaultLocale || '.' || meta.name;
			END IF;
		END LOOP;

	END IF;

	IF isArray THEN
		qs := 'json_agg(' || qs || ')';
	END IF;

	qs := qs || ') AS res FROM {{ $.SchemaName }}.' || tableName || '$' || defaultLocale || suffix || ' ' || tableName || '$' || defaultLocale;

	IF hasLocalized THEN
		qs := qs || ' LEFT JOIN {{ $.SchemaName }}.' || tableName || '$' || locale || suffix || ' ' || tableName || '$' || locale ||
		' ON ' || tableName || '$' || defaultLocale || '.sys_id = ' || tableName || '$' || locale || '.sys_id';
	END IF;

	IF joinedTables IS NOT NULL THEN
		FOREACH meta IN ARRAY joinedTables LOOP
			crit:= {{ $.SchemaName }}._build_critertia(tableName, meta, defaultLocale, locale);
			qs := qs || ' LEFT JOIN LATERAL (' ||
			{{ $.SchemaName }}._include_join(meta.link_type, crit, meta.items_type <> '', locale, defaultLocale, suffix, includeDepth - 1)
			 || ') AS _included_' || meta.name || ' ON true';
		END LOOP;
	END IF;

	IF criteria <> '' THEN
		-- where
		qs := qs || ' WHERE '|| criteria;
	END IF;

	RETURN 'SELECT ' || qs;
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._select_fields(metas {{ $.SchemaName }}._meta[], tableName TEXT, locale TEXT, defaultLocale TEXT, includeDepth INTEGER, suffix TEXT)
RETURNS text AS $$
DECLARE
	qs text:= 'SELECT ';
	hasLocalized boolean := false;
	joinedLaterals text:= '';
	meta {{ $.SchemaName }}._meta;
BEGIN

	-- qs:= qs || tableName || '$' || defaultLocale || '.sys_id  as sys_id,';
	qs := qs || 'json_build_object(''id'','  || tableName || '$' || defaultLocale || '.sys_id) as sys';

	FOREACH meta IN ARRAY metas LOOP
	    qs := qs || ', ';

		-- joins
		IF meta.link_type <> '' AND includeDepth > 0 THEN
			qs := qs || '_included_' || meta.name || '.res';
			joinedLaterals := joinedLaterals || ' LEFT JOIN LATERAL (' ||
			{{ $.SchemaName }}._include_join(meta.link_type, {{ $.SchemaName }}._build_critertia(tableName, meta, defaultLocale, locale), meta.items_type <> '', locale, defaultLocale, suffix, includeDepth - 1) || ') AS _included_' || meta.name || ' ON true';
		ELSEIF meta.is_localized AND locale <> defaultLocale THEN
			qs := qs || 'COALESCE(' || tableName || '$' || locale || '.' || meta.name || ',' ||
			tableName || '$' || defaultLocale || '.' || meta.name || ')';
		ELSE
	    	qs := qs || tableName || '$' || defaultLocale || '.' || meta.name;
		END IF;

		IF meta.is_localized AND locale <> defaultLocale THEN
			hasLocalized := true;
		END IF;

		qs := qs || ' as "' || {{ $.SchemaName }}._fmt_column_name(meta.name) || '"';
	END LOOP;

	qs := qs || ' FROM {{ $.SchemaName }}.' || tableName || '$' || defaultLocale || suffix || ' ' || tableName || '$' || defaultLocale;

	IF hasLocalized THEN
		qs := qs || ' LEFT JOIN {{ $.SchemaName }}.' || tableName || '$' || locale || suffix || ' ' || tableName || '$' || locale ||
		' ON ' || tableName || '$' || defaultLocale || '.sys_id = ' || tableName || '$' || locale || '.sys_id';
	END IF;

	IF joinedLaterals IS NOT NULL THEN
		qs := qs || joinedLaterals;
	END IF;

	RETURN qs;
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._filter_clauses(metas {{ $.SchemaName }}._meta[], tableName TEXT, defaultLocale TEXT, locale TEXT, filters {{ $.SchemaName }}._filter[])
RETURNS text AS $$
DECLARE
	qs text := '';
	filter {{ $.SchemaName }}._filter;
	fFields text[];
	meta {{ $.SchemaName }}._meta;
	clauses text[];
	crit text;
	isFirst boolean := true;
BEGIN
	IF filters IS NOT NULL THEN
		FOREACH filter IN ARRAY filters LOOP
			fFields:= string_to_array(filter.field, '.');
			SELECT * FROM unnest(metas) WHERE name = fFields[1] INTO meta;
			clauses:= clauses || {{ $.SchemaName }}._fmt_clause(meta, tableName, defaultLocale, locale, filter.comparer, filter.values, fFields[1], fFields[2]);
		END LOOP;
	END IF;

	IF clauses IS NOT NULL THEN
		-- where
		FOREACH crit IN ARRAY clauses LOOP
			IF crit <> '' THEN
				IF isFirst THEN
			    	isFirst := false;
					qs := qs || ' WHERE ';
			    ELSE
			    	qs := qs || ' AND ';
			    END IF;
				qs := qs || '(' || crit || ')';
			END IF;
		END LOOP;
	END IF;

	RETURN qs;
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._finalize_query(INOUT qs TEXT, orderBy TEXT, skip INTEGER, take INTEGER, count BOOLEAN)
AS $$
BEGIN
	IF count THEN
		qs:= 'SELECT COUNT(t.sys) as count FROM (' || qs || ') t';
	ELSE
		IF orderBy <> '' THEN
			qs:= qs || ' ORDER BY ' || orderBy;
		END IF;

		IF skip <> 0 THEN
			qs:= qs || ' OFFSET ' || skip;
		END IF;

		IF take <> 0 THEN
			qs:= qs || ' LIMIT ' || take;
		END IF;

		qs:= 'SELECT array_to_json(array_agg(row_to_json(t))) FROM (' || qs || ') t';
	END IF;
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._generate_query(tableName TEXT, locale TEXT, defaultLocale TEXT, fields TEXT[], filters {{ $.SchemaName }}._filter[], orderBy TEXT, skip INTEGER, take INTEGER, includeDepth INTEGER, usePreview BOOLEAN, count BOOLEAN)
RETURNS text AS $$
DECLARE
	qs text;
	suffix text := '$publish';
	metas {{ $.SchemaName }}._meta[];
BEGIN
	IF usePreview THEN
		suffix := '';
	END IF;

	SELECT ARRAY(SELECT {{ $.SchemaName }}._get_meta(tableName)) INTO metas;

	qs := {{ $.SchemaName }}._select_fields(metas, tableName, locale, defaultLocale, includeDepth, suffix);

	qs:= qs || {{ $.SchemaName }}._filter_clauses(metas, tableName, defaultLocale, locale, filters);

	qs := {{ $.SchemaName }}._finalize_query(qs, orderBy, skip, take, count);

	RETURN qs;
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._join_exclude_games(market TEXT, device TEXT, defaultLocale TEXT, suffix TEXT)
RETURNS TEXT AS $$
BEGIN
	RETURN ' LEFT JOIN LATERAL(SELECT array_agg(game_device_configuration.sys_id) AS games_exclude_from_market FROM {{ $.SchemaName }}.games_exclude_from_market$' || defaultLocale || ' games_exclude_from_market LEFT JOIN {{ $.SchemaName }}.game_id$' || defaultLocale ||
	' game_device_configuration ON game_device_configuration.sys_id = ANY(games_exclude_from_market.games) LEFT JOIN {{ $.SchemaName }}.game_device$' || 	defaultLocale || ' AS game_device ON lower(game_device.type) = ''' || device || ''' WHERE games_exclude_from_market.market = ''' ||
	market || ''' AND game_device.sys_id = ANY(game_device_configuration.devices)) AS games_exclude_from_market ON true LEFT JOIN LATERAL(
SELECT studios AS game_studio_exclude_from_market FROM {{ $.SchemaName }}.game_studio_exclude_from_market$' || defaultLocale || ' WHERE market = ''' ||
	market || ''') AS game_studio_exclude_from_market ON true';
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._generate_gamebrowser(market TEXT, device TEXT, tableName TEXT, locale TEXT, defaultLocale TEXT, fields TEXT[], filters {{ $.SchemaName }}._filter[], orderBy TEXT, skip INTEGER, take INTEGER, includeDepth INTEGER, usePreview BOOLEAN, count BOOLEAN)
RETURNS text AS $$
DECLARE
	qs text;
	suffix text := '$publish';
	metas {{ $.SchemaName }}._meta[];
	fc text;
BEGIN
	IF usePreview THEN
		suffix := '';
	END IF;

	SELECT ARRAY(SELECT {{ $.SchemaName }}._get_meta(tableName)) INTO metas;

	qs := {{ $.SchemaName }}._select_fields(metas, tableName, locale, defaultLocale, includeDepth, suffix);

	qs := qs || {{ $.SchemaName }}._join_exclude_games(market, device, defaultLocale, suffix);

	fc := {{ $.SchemaName }}._filter_clauses(metas, tableName, defaultLocale, locale, filters);

	IF fc <> '' THEN
		qs :=  qs || fc || ' AND ';
	ELSE
		qs :=  qs || ' WHERE ';
	END IF;

	qs := qs || '(game_studio_exclude_from_market IS NULL OR ' ||
	tableName || '$' || defaultLocale || '.studio <> ALL(game_studio_exclude_from_market)) AND ' ||
	'(games_exclude_from_market IS NULL OR NOT ' ||
	tableName || '$' || defaultLocale || '.device_configurations && games_exclude_from_market)';

	qs := {{ $.SchemaName }}._finalize_query(qs, orderBy, skip, take, count);

	RETURN qs;
END;
$$ LANGUAGE 'plpgsql';

--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._run_query(tableName TEXT, locale TEXT, defaultLocale TEXT, fields TEXT[], filters {{ $.SchemaName }}._filter[], orderBy TEXT, skip INTEGER, take INTEGER, includeDepth INTEGER, usePreview BOOLEAN)
RETURNS {{ $.SchemaName }}._result AS $$
DECLARE
	count integer;
	items json;
	res {{ $.SchemaName }}._result;
BEGIN
	EXECUTE {{ $.SchemaName }}._generate_query(tableName, locale, defaultLocale, fields, filters, orderBy, skip, take, includeDepth, usePreview, true) INTO count;
	EXECUTE {{ $.SchemaName }}._generate_query(tableName, locale, defaultLocale, fields, filters, orderBy, skip, take, includeDepth, usePreview, false) INTO items;
	IF items IS NULL THEN
		items:= '[]'::JSON;
	END IF;
	RETURN ROW(count, items)::{{ $.SchemaName }}._result;
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._run_query(market TEXT, device TEXT, tableName TEXT, locale TEXT, defaultLocale TEXT, fields TEXT[], filters {{ $.SchemaName }}._filter[], orderBy TEXT, skip INTEGER, take INTEGER, includeDepth INTEGER, usePreview BOOLEAN)
RETURNS {{ $.SchemaName }}._result AS $$
DECLARE
	count integer;
	items json;
	res {{ $.SchemaName }}._result;
BEGIN
	EXECUTE {{ $.SchemaName }}._generate_gamebrowser(market, device, tableName, locale, defaultLocale, fields, filters, orderBy, skip, take, includeDepth, usePreview, true) INTO count;
	EXECUTE {{ $.SchemaName }}._generate_gamebrowser(market, device, tableName, locale, defaultLocale, fields, filters, orderBy, skip, take, includeDepth, usePreview, false) INTO items;
	IF items IS NULL THEN
		items:= '[]'::JSON;
	END IF;
	RETURN ROW(count, items)::{{ $.SchemaName }}._result;
END;
$$ LANGUAGE 'plpgsql';
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._models$history(
	_id serial primary key,
	pub_id integer not null,
	name text not null,
	fields jsonb not null,
	version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null
);
--
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._asset$meta (
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
CREATE UNIQUE INDEX IF NOT EXISTS name ON {{ $.SchemaName }}._asset$meta(name);
--
{{ range $aidx, $col := $.AssetColumns }}
INSERT INTO {{ $.SchemaName }}._asset$meta (
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._asset${{ $locale }} (
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
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON {{ $.SchemaName }}._asset${{ $locale }}(sys_id);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._asset${{ $locale }}$publish (
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
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON {{ $.SchemaName }}._asset${{ $locale }}$publish(sys_id);
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.asset${{ $locale }}_upsert(text, text, text, text, text, text, integer, timestamp, text, timestamp, text) CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.asset${{ $locale }}_upsert(_sysId text, _title text, _description text, _fileName text, _contentType text, _url text, _version integer, _created_at timestamp, _created_by text, _updated_at timestamp, _updated_by text)
RETURNS void AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}._asset${{ $locale }} (
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
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.asset${{ $locale }}_publish(integer) CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.asset${{ $locale }}_publish(_aid integer)
RETURNS void AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}._asset${{ $locale }}$publish (
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
FROM {{ $.SchemaName }}._asset${{ $locale }}
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}$meta (
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
CREATE UNIQUE INDEX IF NOT EXISTS name ON {{ $.SchemaName }}.{{ $tbl.TableName }}$meta(name);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}$meta_history (
	_id serial primary key,
	meta_id integer not null,
	name text not null,
	fields jsonb not null,
	created_at timestamp without time zone default now(),
	created_by text not null
);
--
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
{{ range $fieldsidx, $fields := $tbl.Data.Metas }}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}$meta (
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }} (
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
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}(sys_id);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}$publish (
	_id serial primary key,
	sys_id text not null unique,
	{{- range $colidx, $col := $tbl.Columns }}
	"{{ .ColumnName }}" {{ .ColumnType }}{{ .ColumnDesc }}{{- if and .Required (eq $locale $.DefaultLocale) }} not null{{- end -}},
	{{- end }}
	version integer not null default 0,
	published_at timestamp without time zone not null default now(),
	published_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}$publish(sys_id);
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}_upsert(text,{{ range $colidx, $col := $tbl.Columns }} {{ .ColumnType }},{{ end }} integer, timestamp, text, timestamp, text) CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}_upsert(_sysId text,{{ range $colidx, $col := $tbl.Columns }} _{{ .ColumnName }} {{ .ColumnType }},{{ end }} _version integer, _created_at timestamp, _created_by text, _updated_at timestamp, _updated_by text)
RETURNS void AS $$
BEGIN
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }} (
	sys_id,
	{{- range $colidx, $col := $tbl.Columns }}
	"{{ .ColumnName }}",
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
	"{{ .ColumnName }}" = EXCLUDED.{{ .ColumnName }},
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
	"{{ .ColumnName }}",
	{{- end }}
	version,
	published_by
)
SELECT
	sys_id,
	{{- range $colidx, $col := $tbl.Columns }}
	"{{ .ColumnName }}",
	{{- end }}
	version,
	updated_by
FROM {{ $.SchemaName }}.{{ $tbl.TableName }}${{ $locale }}
WHERE _id = _aid
ON CONFLICT (sys_id) DO UPDATE
SET
	{{- range $colidx, $col := $tbl.Columns }}
	"{{ .ColumnName }}" = EXCLUDED.{{ .ColumnName }},
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
{{ end -}}
COMMIT;
`
