# Go Build Flow

1. `go build ./...` — compile all packages
2. `go vet ./...` — static analysis
3. `go test ./... -count=1 -timeout 120s` — run tests
4. `go test -cover ./...` — check coverage
5. `go mod tidy` — clean dependencies
