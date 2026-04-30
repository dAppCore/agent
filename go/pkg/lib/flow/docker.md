# Docker Build Flow

1. `docker build -t app:local .` — build image
2. `docker run --rm app:local /bin/sh -c "echo ok"` — smoke test
3. `docker compose up -d` — start services
4. `docker compose ps` — verify health
5. `docker compose logs --tail=20` — check logs
