#!/bin/sh

cid=$(docker ps -aqf 'name=postgres')
echo 'DROP SCHEMA IF EXISTS content CASCADE' | docker exec -i $cid psql -U postgres

cd cmd/gfl

echo ''
go run . schema pg -s $CF_SPACE -t $CF_TOKEN --url postgres://postgres@localhost:5432/?sslmode=disable

echo ''
go run . sync pg -s $CF_SPACE -t $CF_TOKEN --url postgres://postgres@localhost:5432/?sslmode=disable

echo ''
echo 'SELECT COUNT(*) FROM content.page_en' | docker exec -i $cid psql -U postgres
