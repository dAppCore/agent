# Core Conventions Checklist

## Error Handling
- [ ] `coreerr.E("pkg.Method", "msg", err)` — always 3 args
- [ ] Never `fmt.Errorf` or `errors.New`
- [ ] Import as `coreerr "forge.lthn.ai/core/go-log"`

## File I/O
- [ ] `coreio.Local.Read/Write/EnsureDir` — never `os.ReadFile/WriteFile`
- [ ] `WriteMode(path, content, 0600)` for sensitive files (keys, hashes)
- [ ] Import as `coreio "forge.lthn.ai/core/go-io"`

## Safety
- [ ] Check `err != nil` BEFORE `resp.StatusCode`
- [ ] Type assertions use comma-ok: `v, ok := x.(Type)`
- [ ] No hardcoded paths (`/Users/`, `/home/`, `host-uk`)
- [ ] No tokens/secrets in error messages or logs

## Style
- [ ] UK English in comments (colour, organisation, initialise)
- [ ] SPDX-License-Identifier: EUPL-1.2 on every file
- [ ] Test naming: `_Good`, `_Bad`, `_Ugly`
