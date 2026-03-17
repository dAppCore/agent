# Release Flow

1. Update version in source files
2. Update CHANGELOG.md
3. `git tag vX.Y.Z` — tag
4. `git push origin main --tags` — push with tags
5. Build release artefacts (platform-specific)
6. Create Forge/GitHub release with artefacts
7. Update downstream go.mod dependencies
