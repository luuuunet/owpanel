package aisite

// gitCloneIntoSiteRootBlock clears the site root (panel may create index.html and pre-seed
// .env during Plan), clones the repo into ".", then restores a stashed .env if present.
func gitCloneIntoSiteRootBlock() string {
	return `ENV_BACKUP=""
if [ -f .env ]; then
  ENV_BACKUP="$(mktemp)"
  cp .env "$ENV_BACKUP"
fi
find . -mindepth 1 -maxdepth 1 -exec rm -rf {} + 2>/dev/null || true
git clone --depth 1 -b "{{branch}}" "{{repo}}" .
if [ -n "$ENV_BACKUP" ] && [ -f "$ENV_BACKUP" ]; then
  cp "$ENV_BACKUP" .env
  rm -f "$ENV_BACKUP"
fi
`
}
