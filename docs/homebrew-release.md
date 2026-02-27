# Homebrew Release Flow

`wtp` Homebrew distribution is managed without GoReleaser `brews`.

## Source of Truth

- Formula template: `packaging/homebrew/wtp.rb.tmpl`
- Render script: `scripts/render-homebrew-formula.sh`
- Sync script: `scripts/sync-homebrew-formula.sh`

## Release Workflow

On tag push (`v*`):

1. GoReleaser publishes release assets and packages.
2. `sync-homebrew-formula.sh` syncs formula structure to `satococoa/homebrew-tap`
   while preserving existing `version` and `sha256`.
3. `mislav/bump-homebrew-formula-action@v3` bumps `version` and `sha256`
   for stable tags only (tags without `-`).

## Updating Homebrew Install Behavior

When changing install/completion behavior:

1. Update `packaging/homebrew/wtp.rb.tmpl`.
2. Validate template rendering:
   - `scripts/render-homebrew-formula.sh --version <version> --sha256 <sha256> --output /tmp/wtp.rb`
3. Let release workflow sync the structure to `homebrew-tap`.
