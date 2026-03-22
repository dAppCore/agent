# PHP Build Flow

1. `composer install --no-interaction` — install deps
2. `./vendor/bin/pint --test` — check formatting (PSR-12)
3. `./vendor/bin/phpstan analyse` — static analysis
4. `./vendor/bin/pest --no-interaction` — run tests
5. `composer audit` — security check
