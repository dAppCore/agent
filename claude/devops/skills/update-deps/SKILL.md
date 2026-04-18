---
name: update-deps
description: This skill should be used when the user asks to "update deps", "bump core", "update go.mod", "upgrade dependencies", or needs to update dappco.re/go/core or other Go module dependencies in a core ecosystem repo. Uses go get properly — never manual go.mod editing.
argument-hint: [repo-name] [module@version]
allowed-tools: ["Bash"]
---

# Update Go Module Dependencies

Properly update dependencies in a Core ecosystem Go module.

## Steps

1. Determine the repo. If an argument is given, use it. Otherwise use the current working directory.
   ```
   /Users/snider/Code/core/<repo>/
   ```

2. Check current dependency versions:
   ```bash
   grep 'dappco.re' go.mod
   ```

3. Update the dependency using `go get`. Examples:
   ```bash
   # Update core to latest
   GONOSUMDB='dappco.re/*' GONOSUMCHECK='dappco.re/*' GOPROXY=direct go get dappco.re/go/core@latest

   # Update to specific version
   GONOSUMDB='dappco.re/*' GONOSUMCHECK='dappco.re/*' GOPROXY=direct go get dappco.re/go/core@v0.6.0

   # Update all dappco.re deps
   GONOSUMDB='dappco.re/*' GONOSUMCHECK='dappco.re/*' GOPROXY=direct go get -u dappco.re/...
   ```

4. Tidy:
   ```bash
   go mod tidy
   ```

5. Verify:
   ```bash
   go build ./...
   ```

6. Report what changed in go.mod.

## Important

- ALWAYS use `go get` — NEVER manually edit go.mod
- ALWAYS set `GONOSUMDB` and `GONOSUMCHECK` for dappco.re modules
- ALWAYS set `GOPROXY=direct` to bypass proxy cache for private modules
- ALWAYS run `go mod tidy` after updating
- ALWAYS verify with `go build ./...`
- If a version doesn't resolve, check if the tag has been pushed to GitHub (dappco.re vanity imports resolve through GitHub)
