#!/bin/sh

sqlite3 ./db/storage.db 'VACUUM;'
sqlite3 ./db/storage.db < ./internal/storage/sqlite/migrations/000001_init_mg.up.sql

./dist/app