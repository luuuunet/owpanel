package aisite

// npmInstallBlock installs JS dependencies using the lockfile / package manager
// that matches the cloned project (pnpm, yarn, or npm).
func npmInstallBlock() string {
	return `if [ ! -f package.json ]; then
  echo "no package.json — skip npm install"
else
  corepack enable 2>/dev/null || true
  echo "node $(node -v 2>/dev/null || echo missing) npm $(npm -v 2>/dev/null || echo missing)"
  if [ -f pnpm-lock.yaml ]; then
    if ! command -v pnpm >/dev/null 2>&1; then
      corepack prepare pnpm@latest --activate 2>/dev/null || npm install -g pnpm@latest
    fi
    pnpm install --frozen-lockfile 2>/dev/null || pnpm install
  elif [ -f yarn.lock ]; then
    if ! command -v yarn >/dev/null 2>&1; then
      corepack prepare yarn@stable --activate 2>/dev/null || npm install -g yarn
    fi
    yarn install --frozen-lockfile 2>/dev/null || yarn install
  elif [ -f package-lock.json ]; then
    npm ci --no-audit --no-fund
  elif grep -qE 'catalog:' package.json 2>/dev/null; then
    NPM_MAJOR=$(npm -v 2>/dev/null | cut -d. -f1)
    NPM_MINOR=$(npm -v 2>/dev/null | cut -d. -f2)
    if [ "${NPM_MAJOR:-0}" -lt 10 ] || { [ "${NPM_MAJOR:-0}" -eq 10 ] && [ "${NPM_MINOR:-0}" -lt 7 ]; }; then
      echo "npm $(npm -v) lacks catalog: support — using npx npm@latest install"
      npx --yes npm@latest install --no-audit --no-fund
    else
      npm install --no-audit --no-fund
    fi
  else
    npm install --no-audit --no-fund
  fi
fi
`
}

// npmBuildBlock runs frontend build with monorepo / env-var awareness.
// Placeholders {{domain_host}} are replaced before execution.
func npmBuildBlock() string {
	return `if [ ! -f package.json ]; then
  echo "no package.json — skip build"
else
  git config --global --add safe.directory "$(pwd)" 2>/dev/null || true
  export NODE_ENV=production
  SITE_DOMAIN="{{domain_host}}"
  export DOCS_ORIGIN="${DOCS_ORIGIN:-https://docs.${SITE_DOMAIN}}"
  export BLOG_ORIGIN="${BLOG_ORIGIN:-https://blog.${SITE_DOMAIN}}"
  export NEXT_PUBLIC_SITE_URL="${NEXT_PUBLIC_SITE_URL:-https://${SITE_DOMAIN}}"
  export SITE_URL="${SITE_URL:-https://${SITE_DOMAIN}}"
  run_pkg_build() {
    local dir="$1"
    local filter="$2"
    if [ -n "$filter" ] && command -v pnpm >/dev/null 2>&1; then
      echo "monorepo build: pnpm --filter ${filter} build"
      pnpm --filter "${filter}" build
    elif [ -n "$dir" ] && [ -f "$dir/package.json" ]; then
      echo "monorepo build: (cd $dir && npm run build)"
      (cd "$dir" && npm run build)
    else
      echo "fallback: npm run build"
      npm run build
    fi
  }
  if [ -f turbo.json ] && [ -d apps ]; then
    if [ -f apps/site/package.json ]; then
      run_pkg_build apps/site site
    elif [ -f apps/web/package.json ]; then
      run_pkg_build apps/web web
    elif [ -f apps/www/package.json ]; then
      run_pkg_build apps/www www
    elif [ -f apps/app/package.json ]; then
      run_pkg_build apps/app app
    else
      FIRST_APP=$(find apps -mindepth 2 -maxdepth 2 -name package.json | head -1)
      if [ -n "$FIRST_APP" ]; then
        APP_DIR=$(dirname "$FIRST_APP")
        APP_NAME=$(basename "$APP_DIR")
        run_pkg_build "$APP_DIR" "$APP_NAME"
      elif grep -q '"build"' package.json; then
        run_pkg_build "" ""
      fi
    fi
  elif grep -q '"build"' package.json; then
    if command -v pnpm >/dev/null 2>&1 && [ -f pnpm-lock.yaml ]; then
      pnpm run build
    else
      npm run build
    fi
  fi
fi
`
}
