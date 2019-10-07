package gontentful

const pgTemplate = `BEGIN;
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
        FROM {{ $.SchemaName }}.' || tableName || '___meta';

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
			RETURN ' IS NOT DISTINCT FROM ' || fmtVal;
		ELSEIF  comparer = 'ne' THEN
			RETURN ' IS DISTINCT FROM ' || fmtVal;
		ELSEIF  comparer = 'exists' THEN
			RETURN ' IS NOT NULL';
		ELSEIF  comparer = 'lt' THEN
			RETURN ' < ' || fmtVal;
		ELSEIF  comparer = 'lte' THEN
			RETURN ' <= ' || fmtVal;
		ELSEIF  comparer = 'gt' THEN
			RETURN ' > ' || fmtVal;
		ELSEIF  comparer = 'gte' THEN
			RETURN ' >= ' || fmtVal;
		ELSEIF comparer = 'match' THEN
			RETURN ' ILIKE ' || fmtVal;
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
			RETURN 'COALESCE(' || tableName || '__' || locale || '.' || meta.name || ',' ||
			tableName || '__' || defaultLocale || '.' || meta.name || ')' || {{ $.SchemaName }}._fmt_comparer(comparer, fmtVal, isArray);
		END IF;
		RETURN tableName || '__' || defaultLocale || '.' || meta.name || {{ $.SchemaName }}._fmt_comparer(comparer, fmtVal, isArray);
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
					fmtVal := fmtVal || 'COALESCE(' || tableName || '__' || locale || '.' || meta.name || ',' ||
					tableName || '__' || defaultLocale || '.' || meta.name || ')' || fmtComp;
				ELSE
					fmtVal := fmtVal || tableName || '__' || defaultLocale || '.' || meta.name || fmtComp;
				END IF;
			ELSE
				fmtVal := fmtVal || tableName || '__' || defaultLocale || '.' || field || fmtComp;
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
	c:= meta.link_type || '__' || defaultLocale || '.sys_id = ';
	IF meta.is_localized AND locale <> defaultLocale THEN
		f := 'COALESCE(' || tableName || '__' || locale || '.' || meta.name || ',' ||
		tableName || '__' || defaultLocale || '.' || meta.name || ')';
	ELSE
		f := tableName || '__' || defaultLocale || '.' || meta.name;
	END IF;

	IF meta.items_type <> '' THEN
		f := 'ANY(' || f || ')';
	END IF;

	RETURN c || f;
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._include_join(tableName TEXT, criteria TEXT, isArray BOOLEAN, locale TEXT, defaultLocale TEXT, includeDepth INTEGER)
RETURNS text AS $$
DECLARE
	qs text;
	hasLocalized boolean := false;
	joinedTables {{ $.SchemaName }}._meta[];
	meta {{ $.SchemaName }}._meta;
	crit text;
BEGIN
	qs := 'json_build_object(';

	-- qs:= qs || tableName || '__' || defaultLocale || '.sys_id as sys_id, ';
	qs:= qs || '''sys'',json_build_object(''id'','  || tableName || '__' || defaultLocale || '.sys_id)';

	IF tableName = '_asset' THEN
		qs := qs || ',';

		IF locale <> defaultLocale THEN
			hasLocalized:= true;
		END IF;

		IF hasLocalized THEN
			qs := qs ||
			'''title'',' || 'COALESCE(' || tableName || '__' || locale || '.title,' || tableName || '__' || defaultLocale || '.title),' ||
			'''description'',' || 'COALESCE(' || tableName || '__' || locale || '.description,' || tableName || '__' || defaultLocale || '.description),' ||
			'''file'',json_build_object(' ||
				'''contentType'',COALESCE(' || tableName || '__' || locale || '.content_type,' || tableName || '__' || defaultLocale || '.content_type),' ||
				'''fileName'',COALESCE(' || tableName || '__' || locale || '.file_name,' || tableName || '__' || defaultLocale || '.file_name),' ||
				'''url'',COALESCE(' || tableName || '__' || locale || '.url,' || tableName || '__' || defaultLocale || '.url))';
		ELSE
			qs := qs ||
			'''title'',' || tableName || '__' || defaultLocale || '.title,' ||
			'''description'',' || tableName || '__' || defaultLocale || '.description,' ||
			'''file'',json_build_object(' ||
				'''contentType'',' || tableName || '__' || defaultLocale || '.content_type,' ||
				'''fileName'',' || tableName || '__' || defaultLocale || '.file_name,' ||
				'''url'',' || tableName || '__' || defaultLocale || '.url)';
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
				qs := qs || 'COALESCE(' || tableName || '__' || locale || '.' || meta.name || ',' ||
					tableName || '__' || defaultLocale || '.' || meta.name || ')';
			ELSE
			   	qs := qs || tableName || '__' || defaultLocale || '.' || meta.name;
			END IF;
		END LOOP;

	END IF;

	IF isArray THEN
		qs := 'json_agg(' || qs || ')';
	END IF;

	qs := qs || ') AS res FROM {{ $.SchemaName }}.' || tableName || '__' || defaultLocale || ' ' || tableName || '__' || defaultLocale;

	IF hasLocalized THEN
		qs := qs || ' LEFT JOIN {{ $.SchemaName }}.' || tableName || '__' || locale || ' ' || tableName || '__' || locale ||
		' ON ' || tableName || '__' || defaultLocale || '.sys_id = ' || tableName || '__' || locale || '.sys_id';
	END IF;

	IF joinedTables IS NOT NULL THEN
		FOREACH meta IN ARRAY joinedTables LOOP
			crit:= {{ $.SchemaName }}._build_critertia(tableName, meta, defaultLocale, locale);
			qs := qs || ' LEFT JOIN LATERAL (' ||
			{{ $.SchemaName }}._include_join(meta.link_type, crit, meta.items_type <> '', locale, defaultLocale, includeDepth - 1)
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
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._select_fields(metas {{ $.SchemaName }}._meta[], tableName TEXT, locale TEXT, defaultLocale TEXT, includeDepth INTEGER)
RETURNS text AS $$
DECLARE
	qs text:= 'SELECT ';
	hasLocalized boolean := false;
	joinedLaterals text:= '';
	meta {{ $.SchemaName }}._meta;
BEGIN

	-- qs:= qs || tableName || '__' || defaultLocale || '.sys_id  as sys_id,';
	qs := qs || 'json_build_object(''id'','  || tableName || '__' || defaultLocale || '.sys_id) as sys';

	FOREACH meta IN ARRAY metas LOOP
	    qs := qs || ', ';

		-- joins
		IF meta.link_type <> '' AND includeDepth > 0 THEN
			qs := qs || '_included_' || meta.name || '.res';
			joinedLaterals := joinedLaterals || ' LEFT JOIN LATERAL (' ||
			{{ $.SchemaName }}._include_join(meta.link_type, {{ $.SchemaName }}._build_critertia(tableName, meta, defaultLocale, locale), meta.items_type <> '', locale, defaultLocale, includeDepth - 1) || ') AS _included_' || meta.name || ' ON true';
		ELSEIF meta.is_localized AND locale <> defaultLocale THEN
			qs := qs || 'COALESCE(' || tableName || '__' || locale || '.' || meta.name || ',' ||
			tableName || '__' || defaultLocale || '.' || meta.name || ')';
		ELSE
	    	qs := qs || tableName || '__' || defaultLocale || '.' || meta.name;
		END IF;

		IF meta.is_localized AND locale <> defaultLocale THEN
			hasLocalized := true;
		END IF;

		qs := qs || ' as "' || {{ $.SchemaName }}._fmt_column_name(meta.name) || '"';
	END LOOP;

	qs := qs || ' FROM {{ $.SchemaName }}.' || tableName || '__' || defaultLocale || ' ' || tableName || '__' || defaultLocale;

	IF hasLocalized THEN
		qs := qs || ' LEFT JOIN {{ $.SchemaName }}.' || tableName || '__' || locale || ' ' || tableName || '__' || locale ||
		' ON ' || tableName || '__' || defaultLocale || '.sys_id = ' || tableName || '__' || locale || '.sys_id';
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
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._generate_query(tableName TEXT, locale TEXT, defaultLocale TEXT, fields TEXT[], filters {{ $.SchemaName }}._filter[], orderBy TEXT, skip INTEGER, take INTEGER, includeDepth INTEGER, count BOOLEAN)
RETURNS text AS $$
DECLARE
	qs text;
	metas {{ $.SchemaName }}._meta[];
BEGIN
	SELECT ARRAY(SELECT {{ $.SchemaName }}._get_meta(tableName)) INTO metas;

	qs := {{ $.SchemaName }}._select_fields(metas, tableName, locale, defaultLocale, includeDepth);

	qs:= qs || {{ $.SchemaName }}._filter_clauses(metas, tableName, defaultLocale, locale, filters);

	qs := {{ $.SchemaName }}._finalize_query(qs, orderBy, skip, take, count);

	RETURN qs;
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._join_exclude_games(market TEXT, device TEXT, defaultLocale TEXT)
RETURNS TEXT AS $$
BEGIN
	RETURN ' LEFT JOIN LATERAL(SELECT array_agg(game_device_configuration.sys_id) AS games_exclude_from_market FROM {{ $.SchemaName }}.games_exclude_from_market__' || defaultLocale || ' games_exclude_from_market LEFT JOIN {{ $.SchemaName }}.game_id__' || defaultLocale ||
	' game_device_configuration ON game_device_configuration.sys_id = ANY(games_exclude_from_market.games) LEFT JOIN {{ $.SchemaName }}.game_device__' || 	defaultLocale || ' AS game_device ON lower(game_device.type) = ''' || device || ''' WHERE games_exclude_from_market.market = ''' ||
	market || ''' AND game_device.sys_id = ANY(game_device_configuration.devices)) AS games_exclude_from_market ON true LEFT JOIN LATERAL(
SELECT studios AS game_studio_exclude_from_market FROM {{ $.SchemaName }}.game_studio_exclude_from_market__' || defaultLocale || ' WHERE market = ''' ||
	market || ''') AS game_studio_exclude_from_market ON true';
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._generate_gamebrowser(market TEXT, device TEXT, tableName TEXT, locale TEXT, defaultLocale TEXT, fields TEXT[], filters {{ $.SchemaName }}._filter[], orderBy TEXT, skip INTEGER, take INTEGER, includeDepth INTEGER, count BOOLEAN)
RETURNS text AS $$
DECLARE
	qs text;
	metas {{ $.SchemaName }}._meta[];
	fc text;
BEGIN
	SELECT ARRAY(SELECT {{ $.SchemaName }}._get_meta(tableName)) INTO metas;

	qs := {{ $.SchemaName }}._select_fields(metas, tableName, locale, defaultLocale, includeDepth);

	qs := qs || {{ $.SchemaName }}._join_exclude_games(market, device, defaultLocale);

	fc := {{ $.SchemaName }}._filter_clauses(metas, tableName, defaultLocale, locale, filters);

	IF fc <> '' THEN
		qs :=  qs || fc || ' AND ';
	ELSE
		qs :=  qs || ' WHERE ';
	END IF;

	qs := qs || '(game_studio_exclude_from_market IS NULL OR ' ||
	tableName || '__' || defaultLocale || '.studio <> ALL(game_studio_exclude_from_market)) AND ' ||
	'(games_exclude_from_market IS NULL OR NOT ' ||
	tableName || '__' || defaultLocale || '.device_configurations && games_exclude_from_market)';

	qs := {{ $.SchemaName }}._finalize_query(qs, orderBy, skip, take, count);

	RETURN qs;
END;
$$ LANGUAGE 'plpgsql';

--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._run_query(tableName TEXT, locale TEXT, defaultLocale TEXT, fields TEXT[], filters {{ $.SchemaName }}._filter[], orderBy TEXT, skip INTEGER, take INTEGER, includeDepth INTEGER)
RETURNS {{ $.SchemaName }}._result AS $$
DECLARE
	count integer;
	items json;
	res {{ $.SchemaName }}._result;
BEGIN
	EXECUTE {{ $.SchemaName }}._generate_query(tableName, locale, defaultLocale, fields, filters, orderBy, skip, take, includeDepth, true) INTO count;
	EXECUTE {{ $.SchemaName }}._generate_query(tableName, locale, defaultLocale, fields, filters, orderBy, skip, take, includeDepth, false) INTO items;
	IF items IS NULL THEN
		items:= '[]'::JSON;
	END IF;
	RETURN ROW(count, items)::{{ $.SchemaName }}._result;
END;
$$ LANGUAGE 'plpgsql';
--
CREATE OR REPLACE FUNCTION {{ $.SchemaName }}._run_query(market TEXT, device TEXT, tableName TEXT, locale TEXT, defaultLocale TEXT, fields TEXT[], filters {{ $.SchemaName }}._filter[], orderBy TEXT, skip INTEGER, take INTEGER, includeDepth INTEGER)
RETURNS {{ $.SchemaName }}._result AS $$
DECLARE
	count integer;
	items json;
	res {{ $.SchemaName }}._result;
BEGIN
	EXECUTE {{ $.SchemaName }}._generate_gamebrowser(market, device, tableName, locale, defaultLocale, fields, filters, orderBy, skip, take, includeDepth, true) INTO count;
	EXECUTE {{ $.SchemaName }}._generate_gamebrowser(market, device, tableName, locale, defaultLocale, fields, filters, orderBy, skip, take, includeDepth, false) INTO items;
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._asset___meta (
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
CREATE UNIQUE INDEX IF NOT EXISTS name ON {{ $.SchemaName }}._asset___meta(name);
--
{{ range $aidx, $col := $.AssetColumns }}
INSERT INTO {{ $.SchemaName }}._asset___meta (
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}._asset__{{ $locale }} (
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
CREATE UNIQUE INDEX IF NOT EXISTS sys_id ON {{ $.SchemaName }}._asset__{{ $locale }}(sys_id);
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on__asset__{{ $locale }}_insert() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on__asset__{{ $locale }}_insert()
RETURNS TRIGGER AS $$
BEGIN
	INSERT INTO {{ $.SchemaName }}._entries (
		sys_id,
		table_name
	) VALUES (
		NEW.sys_id,
		'_asset__{{ $locale }}'
	) ON CONFLICT (sys_id) DO NOTHING;
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}__asset__{{ $locale }}_insert ON {{ $.SchemaName }}._asset__{{ $locale }};
--
CREATE TRIGGER {{ $.SchemaName }}__asset__{{ $locale }}_insert
	AFTER INSERT ON {{ $.SchemaName }}._asset__{{ $locale }}
	FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on__asset__{{ $locale }}_insert();
--
DROP FUNCTION IF EXISTS {{ $.SchemaName }}.on__asset__{{ $locale }}_delete() CASCADE;
--
CREATE FUNCTION {{ $.SchemaName }}.on__asset__{{ $locale }}_delete()
RETURNS TRIGGER AS $$
BEGIN
	DELETE FROM {{ $.SchemaName }}._entries WHERE sys_id = OLD.sys_id AND table_name = '_asset__{{ $locale }}';
	RETURN NULL;
END;
$$ LANGUAGE plpgsql;
--
DROP TRIGGER IF EXISTS {{ $.SchemaName }}__asset__{{ $locale }}_delete ON {{ $.SchemaName }}._asset__{{ $locale }};
--
CREATE TRIGGER {{ $.SchemaName }}__asset__{{ $locale }}_delete
	AFTER DELETE ON {{ $.SchemaName }}._asset__{{ $locale }}
	FOR EACH ROW
	EXECUTE PROCEDURE {{ $.SchemaName }}.on__asset__{{ $locale }}_delete();
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
	"{{ .ColumnName }}" {{ .ColumnType }},
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
{{ end -}}
{{ end -}}
COMMIT;
`
