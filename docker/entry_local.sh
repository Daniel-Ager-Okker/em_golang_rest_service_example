#!/bin/sh

sqlite3 /rest_service_example/db/storage.db 'VACUUM;'
sqlite3 /rest_service_example/db/storage.db < /rest_service_example/internal/storage/sqlite/migrations/000001_init_mg.up.sql

/rest_service_example/dist/app