# Collection Hooks

Event-driven hooks that trigger during data collection.

## Available Hooks

| Hook | Trigger | Purpose |
|------|---------|---------|
| `collect-whitepaper.sh` | PDF/paper URL detected | Auto-queue whitepapers |
| `on-github-release.sh` | Release found | Archive release metadata |
| `on-explorer-block.sh` | Block data fetched | Index blockchain data |

## Hook Events

### `on_url_found`
Fired when a new URL is discovered during collection.

```bash
# Pattern matching
*.pdf              → collect-whitepaper.sh
*/releases/*       → on-github-release.sh
*/api/block/*      → on-explorer-block.sh
```

### `on_file_collected`
Fired after a file is successfully downloaded.

```bash
# Post-processing
*.json             → validate-json.sh
*.html             → extract-links.sh
*.pdf              → extract-metadata.sh
```

### `on_collection_complete`
Fired when a job batch finishes.

```bash
# Reporting
→ generate-index.sh
→ update-registry.sh
```

## Plugin Integration

For the marketplace plugin system:

```json
{
  "name": "whitepaper-collector",
  "version": "1.0.0",
  "hooks": {
    "on_url_found": {
      "pattern": "*.pdf",
      "handler": "./collect-whitepaper.sh"
    }
  }
}
```

## Registration

Hooks register in `hooks.json`:

```json
{
  "on_url_found": [
    {
      "pattern": "\\.pdf$",
      "handler": "./hooks/collect-whitepaper.sh",
      "priority": 10
    }
  ]
}
```

## Usage in Collectors

Collectors call hooks via:

```bash
# In job-collector/process.sh
source ./hooks/dispatch.sh

# When URL found
dispatch_hook "on_url_found" "$URL"

# When file collected
dispatch_hook "on_file_collected" "$FILE" "$TYPE"
```
