# Wallet Releases Collector

Archive wallet software releases, changelogs, and binary checksums.

## Data Available

| Data Type | Source | Notes |
|-----------|--------|-------|
| Release binaries | GitHub releases | Preserve before deletion |
| Changelogs | Release notes | Feature history |
| Checksums | Release page | Verify integrity |
| Source tags | Git tags | Build from source |

## Usage

```bash
# Collect all releases for a project
./generate-jobs.sh LetheanNetwork/lethean > jobs.txt

# Just metadata (no binaries)
./generate-jobs.sh LetheanNetwork/lethean --metadata-only > jobs.txt

# Include pre-releases
./generate-jobs.sh LetheanNetwork/lethean --include-prereleases > jobs.txt
```

## Output

```
releases-lethean/
├── v5.0.0/
│   ├── release.json          # GitHub API response
│   ├── CHANGELOG.md          # Release notes
│   ├── checksums.txt         # SHA256 of binaries
│   └── assets.json           # Binary URLs (not downloaded)
├── v4.0.1/
│   └── ...
└── INDEX.md                   # Version timeline
```

## Job Format

```
URL|FILENAME|TYPE|METADATA
https://api.github.com/repos/LetheanNetwork/lethean/releases|releases-lethean-all.json|github-api|project=lethean
https://github.com/LetheanNetwork/lethean/releases/tag/v5.0.0|releases-lethean-v5.0.0.html|github-web|project=lethean,version=v5.0.0
```

## Preservation Priority

1. **Critical**: Changelogs, checksums, version numbers
2. **Important**: Release dates, asset lists, download counts
3. **Optional**: Binary downloads (large, reproducible from source)

## Notes

- Abandoned projects often delete releases first
- GitHub API rate limited - use authenticated requests
- Some projects use different release platforms (SourceForge, own CDN)
- Track gpg signature files when available
