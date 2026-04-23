# Fleet HTTPS Certificate Audit - 2026-04-23

## Verdict

**OK**

Fleet registration already goes through a TLS-validating `http.Client`; no production code in `pkg/agentic` overrides TLS verification on the `/v1/fleet/register` path. The audit added regression coverage so this path now fails loudly if certificate verification is bypassed or broken.

## What was checked

- Fleet registration is implemented by `handleFleetRegister`, which builds the registration payload and posts it to `/v1/fleet/register` via `platformPayload` at `pkg/agentic/platform.go:199`, `pkg/agentic/platform.go:210`, and `pkg/agentic/platform.go:221`.
- `platformPayload` sends that request through `HTTPDo` with a Bearer token and the platform base URL from `syncAPIURL()` at `pkg/agentic/platform.go:558`, `pkg/agentic/platform.go:569`, and `pkg/agentic/sync.go:252`.
- `HTTPDo` delegates to `httpDo`, and `httpDo` executes the request with `defaultClient.Do(request)` at `pkg/agentic/transport.go:99`, `pkg/agentic/transport.go:139`, and `pkg/agentic/transport.go:161`.
- The only shared production client on this path is `defaultClient`, defined as `&http.Client{Timeout: 30 * time.Second}` with no custom transport or TLS override at `pkg/agentic/transport.go:13`.

## Regression coverage added

- `testDefaultClientWithTrustedServerCert` now builds a client that trusts only the test server certificate via `RootCAs`, and it explicitly asserts `InsecureSkipVerify` stays `false` at `pkg/agentic/platform_test.go:20` and `pkg/agentic/platform_test.go:28`.
- `TestPlatform_HandleFleetRegister_Good_TrustedTLS` proves the real fleet registration path succeeds against a TLS endpoint when the certificate is trusted by the client at `pkg/agentic/platform_test.go:104`, `pkg/agentic/platform_test.go:114`, and `pkg/agentic/platform_test.go:121`.
- `TestPlatform_HandleFleetRegister_Bad_UntrustedTLSCert` proves the same registration path rejects an untrusted certificate, never reaches the handler, and returns a wrapped error instead of succeeding silently at `pkg/agentic/platform_test.go:131`, `pkg/agentic/platform_test.go:144`, `pkg/agentic/platform_test.go:145`, and `pkg/agentic/platform_test.go:149`.

## Test run

- `go test -mod=mod ./pkg/agentic/...` passed in a temp workspace that preserved the repo's `../mcp` replace layout.
