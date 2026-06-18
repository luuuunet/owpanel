package appstore

// PgExtensionCatalogItem describes a PostgreSQL extension installable via the software store stack.
type PgExtensionCatalogItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Contrib     bool   `json:"contrib"`
	AptPkgFmt   string `json:"-"` // e.g. postgresql-%s-postgis-3
	DnfPkgFmt   string `json:"-"`
}

// PostgreSQLExtensionCatalog — extensions delivered by postgresql-contrib or extra OS packages
// when PostgreSQL is installed from the panel software store.
var PostgreSQLExtensionCatalog = []PgExtensionCatalogItem{
	{Name: "uuid-ossp", Description: "生成 UUID", Contrib: true},
	{Name: "pgcrypto", Description: "加密函数", Contrib: true},
	{Name: "hstore", Description: "键值对存储类型", Contrib: true},
	{Name: "citext", Description: "大小写不敏感文本类型", Contrib: true},
	{Name: "pg_trgm", Description: "模糊文本搜索与相似度", Contrib: true},
	{Name: "btree_gin", Description: "GIN 索引 B-tree 等价操作", Contrib: true},
	{Name: "btree_gist", Description: "GiST 索引 B-tree 等价操作", Contrib: true},
	{Name: "bloom", Description: "Bloom 索引访问方法", Contrib: true},
	{Name: "cube", Description: "多维立方体数据类型", Contrib: true},
	{Name: "seg", Description: "浮点区间数据类型", Contrib: true},
	{Name: "ltree", Description: "层级树结构标签", Contrib: true},
	{Name: "isn", Description: "国际商品编号类型", Contrib: true},
	{Name: "intarray", Description: "整数数组与索引", Contrib: true},
	{Name: "fuzzystrmatch", Description: "字符串模糊匹配", Contrib: true},
	{Name: "unaccent", Description: "去除重音符号", Contrib: true},
	{Name: "tablefunc", Description: "表函数（crosstab 等）", Contrib: true},
	{Name: "earthdistance", Description: "地球表面距离计算", Contrib: true},
	{Name: "dblink", Description: "跨库连接查询", Contrib: true},
	{Name: "postgres_fdw", Description: "外部 PostgreSQL 数据源", Contrib: true},
	{Name: "file_fdw", Description: "外部平面文件数据源", Contrib: true},
	{Name: "pg_stat_statements", Description: "SQL 执行统计", Contrib: true},
	{Name: "pg_buffercache", Description: "共享缓冲区检查", Contrib: true},
	{Name: "pg_prewarm", Description: "预热共享缓冲区", Contrib: true},
	{Name: "pg_freespacemap", Description: "空闲空间映射检查", Contrib: true},
	{Name: "pg_visibility", Description: "可见性映射检查", Contrib: true},
	{Name: "pg_walinspect", Description: "WAL 检查工具", Contrib: true},
	{Name: "pgrowlocks", Description: "行级锁查看", Contrib: true},
	{Name: "pgstattuple", Description: "元组级统计", Contrib: true},
	{Name: "pageinspect", Description: "底层页面检查", Contrib: true},
	{Name: "amcheck", Description: "B-tree 索引完整性检查", Contrib: true},
	{Name: "pg_surgery", Description: "堆元组手术修复", Contrib: true},
	{Name: "dict_int", Description: "整数全文检索字典", Contrib: true},
	{Name: "dict_xsyn", Description: "同义词全文检索字典", Contrib: true},
	{Name: "lo", Description: "大对象管理", Contrib: true},
	{Name: "xml2", Description: "XPath 与 XSLT 查询", Contrib: true},
	{Name: "tcn", Description: "触发器变更通知", Contrib: true},
	{Name: "tsm_system_rows", Description: "TABLESAMPLE 按行采样", Contrib: true},
	{Name: "tsm_system_time", Description: "TABLESAMPLE 按时间采样", Contrib: true},
	{Name: "auto_explain", Description: "自动记录慢查询执行计划", Contrib: true},
	{Name: "postgis", Description: "PostGIS 地理空间扩展", AptPkgFmt: "postgresql-%s-postgis-3", DnfPkgFmt: "postgis33_%s"},
	{Name: "postgis_topology", Description: "PostGIS 拓扑扩展", AptPkgFmt: "postgresql-%s-postgis-3", DnfPkgFmt: "postgis33_%s"},
	{Name: "postgis_raster", Description: "PostGIS 栅格扩展", AptPkgFmt: "postgresql-%s-postgis-3", DnfPkgFmt: "postgis33_%s"},
	{Name: "vector", Description: "pgvector 向量检索", AptPkgFmt: "postgresql-%s-pgvector", DnfPkgFmt: "pgvector_%s"},
	{Name: "pg_repack", Description: "在线表重组与膨胀清理", AptPkgFmt: "postgresql-%s-repack", DnfPkgFmt: "pg_repack_%s"},
	{Name: "hypopg", Description: "假设索引分析", AptPkgFmt: "postgresql-%s-hypopg"},
}

func PostgreSQLExtensionCatalogMap() map[string]PgExtensionCatalogItem {
	m := make(map[string]PgExtensionCatalogItem, len(PostgreSQLExtensionCatalog))
	for _, item := range PostgreSQLExtensionCatalog {
		if _, ok := m[item.Name]; !ok {
			m[item.Name] = item
		}
	}
	return m
}
