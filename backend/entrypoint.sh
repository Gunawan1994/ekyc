#!/bin/sh
set -e

echo "Running database migrations..."

if ! command -v migrate > /dev/null 2>&1; then
    echo "migrate not found, downloading..."
    curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
    mv migrate /usr/local/bin/migrate
fi

migrate -path ./migrations -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}" up

echo "Starting server..."
exec ./server
