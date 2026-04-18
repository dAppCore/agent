---
name: app-split
description: This skill should be used when the user asks to "split an app", "fork an app", "create a new app from host.uk.com", "de-hostuk", "copy app to new domain", or needs to extract a Website module from the host.uk.com monolith into a standalone CorePHP application. Covers the full copy-strip-rebrand process.
---

# App Split — Extract CorePHP App from Monolith

Split a Website module from the host.uk.com monolith into a standalone CorePHP application. The approach is copy-everything-then-strip rather than build-from-scratch.

## When to Use

- Extracting a domain-specific app (lthn.ai, bio.host.uk.com, etc.) from host.uk.com
- Creating a new standalone CorePHP app from the existing platform
- Any "fork and specialise" operation on the host.uk.com codebase

## Process

### 1. Inventory — Decide What Stays and Goes

Before copying, map which modules belong to the target app.

**Inputs needed from user:**
- Target domain (e.g. `lthn.ai`)
- Which `Website/*` modules to keep (check `$domains` in each Boot.php)
- Which `Mod/*` modules to keep (product modules vs platform modules)
- Which `Service/*` providers to keep (depends on kept Mod modules)

Run the inventory script to see all modules and their domain bindings:

```bash
!`scripts/inventory.sh /Users/snider/Code/lab/host.uk.com`
```

Consult `references/module-classification.md` for the standard keep/remove classification.

### 2. Copy — Wholesale Clone

```bash
rsync -a \
  --exclude='vendor/' \
  --exclude='node_modules/' \
  --exclude='.git/' \
  --exclude='storage/logs/*' \
  --exclude='storage/framework/cache/*' \
  --exclude='storage/framework/sessions/*' \
  --exclude='storage/framework/views/*' \
  SOURCE/ TARGET/
```

Copy everything. Do not cherry-pick — the framework has deep cross-references and it is faster to remove than to reconstruct.

### 3. Strip — Remove Unwanted Modules

Delete removed module directories:
```bash
# Website modules
rm -rf TARGET/app/Website/{Host,Html,Lab,Service}

# Mod modules
rm -rf TARGET/app/Mod/{Links,Social,Trees,Front,Hub}

# Service providers that depend on removed Mod modules
rm -rf TARGET/app/Service/Hub
```

### 4. Update Boot.php Providers

Edit `TARGET/app/Boot.php`:
- Remove all `\Website\*\Boot::class` entries for deleted Website modules
- Remove all `\Mod\*\Boot::class` entries for deleted Mod modules
- Remove all `\Service\*\Boot::class` entries for deleted Service providers
- Update class docblock (name, description)
- Update `guestRedirectUrl()` — change fallback login host from `host.uk.com` to target domain

### 5. Rebrand — Domain References

Run the domain scan script to find all references:

```bash
!`scripts/domain-scan.sh TARGET`
```

**Critical files to update** (in priority order):

| File | What to Change |
|------|----------------|
| `composer.json` | name, description, licence |
| `config/app.php` | `base_domain` default |
| `.env.example` | APP_URL, SESSION_DOMAIN, MCP_DOMAIN, DB_DATABASE, mail |
| `vite.config.js` | dev server host + HMR host |
| `app/Boot.php` | providers, guest redirect, comments |
| `CLAUDE.md` | Full rewrite for new app |
| `.gitignore` | Add any env files with secrets |
| `robots.txt` | Sitemap URL, allowed paths |
| `public/errors/*.html` | Support contact links |
| `public/js/*.js` | API base URLs in embed widgets |
| `config/cdn.php` | default_domain, apex URL |
| `config/mail.php` | contact_recipient |
| `database/seeders/` | email, domains, branding |

**Leave alone** (shared infrastructure):
- `analytics.host.uk.com` references in CSP headers and tracking pixels
- CDN storage zone names (same Hetzner/BunnyCDN buckets)
- External links to host.uk.com in footers (legitimate cross-links)

### 6. Secure — Check for Secrets

Scan for env files with real credentials before committing:

```bash
# Find env files that might have secrets
find TARGET -name ".env*" -not -name ".env.example" | while read f; do
  if grep -qE '(KEY|SECRET|PASSWORD|TOKEN)=.{8,}' "$f"; then
    echo "SECRETS: $f — add to .gitignore"
  fi
done
```

### 7. Init Git and Verify

```bash
cd TARGET
git init
git add -A
git status  # Review what's being committed
```

Check for:
- No `.env` files with real secrets staged
- No `auth.json` staged
- No `vendor/` or `node_modules/` staged

## Gotchas

- **Service providers reference Mod modules**: If `Service/Hub` depends on `Mod/Hub` and you remove `Mod/Hub`, also remove `Service/Hub` — otherwise the app crashes on boot.
- **Boot.php $providers is the master list**: Every module must be listed here. Missing entries = module doesn't load. Extra entries for deleted modules = crash.
- **Seeders reference removed services**: SystemUserSeeder sets up analytics, trust, push, bio etc. The seeder uses `class_exists()` checks so it gracefully skips missing services, but domain references still need updating.
- **Composer deps for removed modules**: Packages like `core/php-plug-social` are only needed for removed modules. Safe to remove from composer.json but not urgent — they're just unused.
- **The `.env.lthn-ai` pattern**: Production env files often live in the repo for reference but MUST be gitignored since they contain real credentials.
