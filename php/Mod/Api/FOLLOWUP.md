# API Follow-Up

Foundation delivered in this slice:
- `ApiKey`, `WebhookEndpoint`, and `WebhookDelivery` models with root migrations.
- `WebhookService::dispatch()` wrapped in `DB::transaction()` with queued jobs using `->afterCommit()`.
- `DeliverWebhookJob`, `WebhookSignature`, `RateLimitService`, and API key middleware with Sanctum fallback.
- New `Boot` event listener for `ApiRoutesRegistering`.
- Canonical controller split: `DocsController` for public work and `DocumentationController` for protected admin work.

Remaining RFC work:
- Register the new API module provider in the package entry point so the nested module boots without explicit test registration.
- Build the REST surface: webhook CRUD, API key CRUD, delivery inspection, retry endpoints, and gateway controllers.
- Wire real documentation views, OpenAPI generation, and protected admin docs routes.
- Add rate-limit middleware integration, response headers, and per-endpoint policy wiring on the route layer.
- Extend webhook delivery operations with queue maintenance, replay tooling, and the remaining backoff policy edge cases.
- Add broader coverage for middleware auth flows, docs protection, and end-to-end queue delivery.
