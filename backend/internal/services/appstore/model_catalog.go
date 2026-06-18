package appstore

import "github.com/open-panel/open-panel/internal/services/modelcatalog"

func resolveCatalogEntry(id string) *modelcatalog.ModelCatalogEntry {
	return modelcatalog.ResolveEntry(id)
}
