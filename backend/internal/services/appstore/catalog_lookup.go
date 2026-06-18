package appstore

// CatalogByKey returns store catalog entries indexed by app key.
func CatalogByKey() map[string]catalogItem {
	out := make(map[string]catalogItem, len(catalog))
	for _, item := range mergedCatalog() {
		out[item.Key] = item
	}
	return out
}

// DockerContainerName returns the Docker container name for a store app, if any.
func DockerContainerName(key string) (string, bool) {
	spec, ok := dockerSpec(key)
	if !ok {
		return "", false
	}
	return spec.Container, true
}
