#!/bin/bash
set -e

cd /app

# Wait for MariaDB
until php artisan db:monitor --databases=mariadb 2>/dev/null; do
    echo "[entrypoint] Waiting for MariaDB..."
    sleep 2
done

# Run migrations
php artisan migrate --force --no-interaction

# Cache config/routes/views
php artisan config:cache
php artisan route:cache
php artisan view:cache
php artisan event:cache

# Storage link
php artisan storage:link 2>/dev/null || true

echo "[entrypoint] Laravel ready"
