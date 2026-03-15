# Core Agent — Multistage Dockerfile
# Builds the Laravel app with FrankenPHP + Octane
#
# Build context must be the repo root (..):
#   docker build -f docker/Dockerfile ..

# ============================================================
# Stage 1: PHP Dependencies
# ============================================================
FROM composer:latest AS deps

WORKDIR /build
COPY composer.json composer.lock ./
COPY packages/ packages/
RUN composer install --no-dev --no-scripts --no-autoloader --prefer-dist

COPY . .
RUN composer dump-autoload --optimize

# ============================================================
# Stage 2: Frontend Build
# ============================================================
FROM node:22-alpine AS frontend

WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
COPY --from=deps /build/vendor vendor
RUN npm run build

# ============================================================
# Stage 3: Runtime
# ============================================================
FROM dunglas/frankenphp:1-php8.5-trixie

RUN install-php-extensions \
    pcntl pdo_mysql redis gd intl zip \
    opcache bcmath exif sockets

RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends \
    supervisor curl mariadb-client \
    && rm -rf /var/lib/apt/lists/*

RUN mv "$PHP_INI_DIR/php.ini-production" "$PHP_INI_DIR/php.ini"

WORKDIR /app

# Copy built application
COPY --from=deps --chown=www-data:www-data /build /app
COPY --from=frontend /build/public/build /app/public/build

# Config files
COPY docker/config/octane.ini $PHP_INI_DIR/conf.d/octane.ini
COPY docker/config/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY docker/scripts/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Clear build caches
RUN rm -rf bootstrap/cache/*.php \
    storage/framework/cache/data/* \
    storage/framework/sessions/* \
    storage/framework/views/* \
    && php artisan package:discover --ansi

ENV OCTANE_PORT=8088
EXPOSE 8088 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:${OCTANE_PORT}/up || exit 1

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
