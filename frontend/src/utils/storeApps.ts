/** Resolve install API key from a (possibly grouped) store card and selected version. */
export function resolveInstallKey(app: { key: string; grouped?: boolean; version_entries?: { key: string; version: string }[] }, version: string): string {
  if (!app.grouped || !app.version_entries?.length) return app.key
  const entry = app.version_entries.find(e => e.version === version)
  return entry?.key || app.key
}

export function versionChoices(app: { versions?: string; version_entries?: { version: string }[] }): string[] {
  if (app.version_entries?.length) {
    return app.version_entries.map(e => e.version)
  }
  if (!app.versions) return []
  return app.versions.split(',').map(v => v.trim()).filter(Boolean)
}

export function installedVersionEntries(app: { version_entries?: { version: string; installed: boolean; status?: string }[] }) {
  return (app.version_entries || []).filter(e => e.installed)
}

export function isVersionInstalled(app: { version_entries?: { version: string; installed: boolean }[] }, version: string): boolean {
  return (app.version_entries || []).some(e => e.version === version && e.installed)
}

/** Localized app description (backend description_en + locale fallbacks). */
export function appDescription(
  app: { description?: string; description_en?: string; family_key?: string; grouped?: boolean },
  locale: string,
  t: (key: string) => string,
): string {
  const isEn = locale.startsWith('en')
  if (isEn && app.description_en) return app.description_en
  if (app.family_key) {
    const key = `software.family.${app.family_key}.desc`
    const tr = t(key)
    if (tr && tr !== key) return tr
  }
  if (isEn && app.description && !/[\u4e00-\u9fff]/.test(app.description)) {
    return app.description
  }
  if (isEn) {
    return t('software.descriptionFallback')
  }
  return app.description || ''
}

export function displayAppName(app: { name: string; grouped?: boolean; family_key?: string }, t: (key: string) => string): string {
  if (app.grouped && app.family_key) {
    const key = `software.family.${app.family_key}.name`
    const tr = t(key)
    if (tr !== key) return tr
  }
  return app.name
}

export function iconKeyForApp(app: { key: string; grouped?: boolean; family_key?: string }): string {
  if (app.grouped && app.family_key) return app.family_key
  return app.key
}
