package appstore

func init() {
	for k, v := range dockerAppSpecsExtra {
		dockerAppSpecs[k] = v
	}
	// 兼容旧 key
	if spec, ok := dockerAppSpecs["wordpress-app"]; ok {
		dockerAppSpecs["wordpress"] = spec
	}
}

// dockerAppSpecsExtra — 内置扩展 Docker 部署规格（与 catalogExtraApps 对应）
var dockerAppSpecsExtra = map[string]dockerAppSpec{
	// AI
	"hermes-agent": {Container: "open-panel-hermes", Image: "nousresearch/hermes-agent:latest", Port: "8010:8010"},
	"openclaw":     {Container: "open-panel-openclaw", Image: "openclaw/openclaw:latest", Port: "8011:8011"},
	"qwenpow":      {Container: "open-panel-qwenpow", Image: "qwenlm/qwen-agent:latest", Port: "8012:8012"},
	"maxkb":        {Container: "open-panel-maxkb", Image: "1panel/maxkb:latest", Port: "8080:8080", Volumes: []string{"open-panel-maxkb:/opt/maxkb"}},
	"clawswarm":    {Container: "open-panel-clawswarm", Image: "openclaw/clawswarm:latest", Port: "8013:8013"},
	"upage":        {Container: "open-panel-upage", Image: "ghcr.io/upage/upage:latest", Port: "8014:3000"},
	"dbhub":        {Container: "open-panel-dbhub", Image: "bytebase/dbhub:latest", Port: "8015:8080"},
	"lobe-chat":    {Container: "open-panel-lobe-chat", Image: "lobehub/lobe-chat:latest", Port: "3210:3210", Env: []string{"OPENAI_API_KEY=sk-"}},
	"one-api":      {Container: "open-panel-one-api", Image: "justsong/one-api:latest", Port: "3000:3000", Volumes: []string{"open-panel-one-api:/data"}},
	"new-api":      {Container: "open-panel-new-api", Image: "calciumion/new-api:latest", Port: "3008:3000", Volumes: []string{"open-panel-new-api:/data"}},
	"chatgpt-next-web": {Container: "open-panel-next-web", Image: "yidadaa/chatgpt-next-web:latest", Port: "3009:3000"},
	"milvus":       {Container: "open-panel-milvus", Image: "milvusdb/milvus:latest", Port: "19530:19530", Volumes: []string{"open-panel-milvus:/var/lib/milvus"}},
	"qdrant":       {Container: "open-panel-qdrant", Image: "qdrant/qdrant:latest", Port: "6333:6333", Volumes: []string{"open-panel-qdrant:/qdrant/storage"}},
	"flowise":      {Container: "open-panel-flowise", Image: "flowiseai/flowise:latest", Port: "3010:3000", Volumes: []string{"open-panel-flowise:/root/.flowise"}},

	// 网站 / DevOps / 安全
	"wordpress":            {Container: "open-panel-wordpress", Image: "wordpress:6.7-php8.2-apache", Port: "8081:80", Env: []string{"WORDPRESS_DB_HOST=host.docker.internal"}},
	"jumpserver":           {Container: "open-panel-jumpserver", Image: "jumpserver/jms_all:latest", Port: "8088:80", Volumes: []string{"open-panel-jms:/opt/jumpserver/data"}},
	"nginx-proxy-manager":  {Container: "open-panel-npm", Image: "jc21/nginx-proxy-manager:latest", Port: "81:81", Volumes: []string{"open-panel-npm-data:/data", "open-panel-npm-letsencrypt:/etc/letsencrypt"}},
	"tomcat":               {Container: "open-panel-tomcat", Image: "tomcat:10-jdk17", Port: "8080:8080"},
	"dashy":                {Container: "open-panel-dashy", Image: "lissy93/dashy:latest", Port: "8083:8080", Volumes: []string{"open-panel-dashy:/app/user-data"}},
	"discourse":            {Container: "open-panel-discourse", Image: "discourse/discourse:latest", Port: "8092:80"},
	"bytebase":             {Container: "open-panel-bytebase", Image: "bytebase/bytebase:latest", Port: "8085:8080", Volumes: []string{"open-panel-bytebase:/var/opt/bytebase"}},
	"nocobase":             {Container: "open-panel-nocobase", Image: "nocobase/nocobase:latest", Port: "8084:80", Volumes: []string{"open-panel-nocobase:/app/nocobase/storage"}},
	"verdaccio":            {Container: "open-panel-verdaccio", Image: "verdaccio/verdaccio:latest", Port: "4873:4873", Volumes: []string{"open-panel-verdaccio:/verdaccio/storage"}},
	"opengist":             {Container: "open-panel-opengist", Image: "ghcr.io/opengist/opengist:latest", Port: "8090:6157", Volumes: []string{"open-panel-opengist:/opengist"}},

	// 监控 / 工具
	"zabbix-server": {Container: "open-panel-zabbix", Image: "zabbix/zabbix-server-mysql:ubuntu-6.4-latest", Port: "10051:10051"},
	"zabbix-agent":  {Container: "open-panel-zabbix-agent", Image: "zabbix/zabbix-agent2:ubuntu-6.4-latest", Port: "10050:10050", Env: []string{"ZBX_HOSTNAME=open-panel"}},
	"sftpgo":        {Container: "open-panel-sftpgo", Image: "drakkan/sftpgo:latest", Port: "8082:8080", Volumes: []string{"open-panel-sftpgo:/srv/sftpgo"}},
	"synapse":       {Container: "open-panel-synapse", Image: "matrixdotorg/synapse:latest", Port: "8008:8008", Volumes: []string{"open-panel-synapse:/data"}},
	"dockge":        {Container: "open-panel-dockge", Image: "louislam/dockge:1", Port: "5001:5001", Volumes: []string{"/var/run/docker.sock:/var/run/docker.sock", "open-panel-dockge:/app/data", "open-panel-dockge-stacks:/opt/stacks"}},
	"meilisearch":   {Container: "open-panel-meilisearch", Image: "getmeili/meilisearch:latest", Port: "7700:7700", Volumes: []string{"open-panel-meilisearch:/meili_data"}, Env: []string{"MEILI_MASTER_KEY=openpanel123"}},
	"ntfy":          {Container: "open-panel-ntfy", Image: "binwiederhier/ntfy:latest", Port: "8086:80", Volumes: []string{"open-panel-ntfy:/var/lib/ntfy"}},
	"openvpn":       {Container: "open-panel-openvpn", Image: "kylemanna/openvpn:latest", Port: "1194:1194/udp", Volumes: []string{"open-panel-openvpn:/etc/openvpn"}},
	"watchtower":    {Container: "open-panel-watchtower", Image: "containrrr/watchtower:latest", Volumes: []string{"/var/run/docker.sock:/var/run/docker.sock"}},
	"homepage":      {Container: "open-panel-homepage", Image: "ghcr.io/gethomepage/homepage:latest", Port: "8089:3000", Volumes: []string{"open-panel-homepage:/app/config", "/var/run/docker.sock:/var/run/docker.sock:ro"}},
	"redpanda-console": {Container: "open-panel-redpanda-console", Image: "redpandadata/console:latest", Port: "8091:8080"},
	"obsidian-livesync": {Container: "open-panel-livesync", Image: "vrtmrz/obsidian-livesync-server:latest", Port: "8093:5984", Volumes: []string{"open-panel-livesync:/usr/local/var/lib/couchdb"}},
	"screego":       {Container: "open-panel-screego", Image: "screego/server:latest", Port: "8096:5050"},
	"cloudflared":   {Container: "open-panel-cloudflared", Image: "cloudflare/cloudflared:latest", Env: []string{"TUNNEL_TOKEN="}},
	"beszel":        {Container: "open-panel-beszel", Image: "henrygd/beszel:latest", Port: "8099:8090", Volumes: []string{"open-panel-beszel:/beszel_data"}},
	"wallos":        {Container: "open-panel-wallos", Image: "bellamy/wallos:latest", Port: "8100:80", Volumes: []string{"open-panel-wallos:/var/www/html"}},
	"teable":        {Container: "open-panel-teable", Image: "teableio/teable:latest", Port: "8102:3000", Volumes: []string{"open-panel-teable:/app/.assets"}},
	"rustfs":        {Container: "open-panel-rustfs", Image: "rustfs/rustfs:latest", Port: "8105:9000", Volumes: []string{"open-panel-rustfs:/data"}},
	"n8n":           {Container: "open-panel-n8n", Image: "n8nio/n8n:latest", Port: "5678:5678", Volumes: []string{"open-panel-n8n:/home/node/.n8n"}},
	"activepieces":  {Container: "open-panel-activepieces", Image: "activepieces/activepieces:latest", Port: "5680:80", Volumes: []string{"open-panel-ap:/root/.activepieces"}},
	"rustdesk":      {Container: "open-panel-rustdesk-hb", Image: "rustdesk/rustdesk-server:latest", Port: "21115:21115", Volumes: []string{"open-panel-rustdesk:/root"}},
	"adguard-home":  {Container: "open-panel-adguard", Image: "adguard/adguardhome:latest", Port: "3011:3000", Volumes: []string{"open-panel-adguard:/opt/adguardhome/work"}},
	"stirling-pdf":  {Container: "open-panel-stirling", Image: "frooodle/s-pdf:latest", Port: "3016:8080", Volumes: []string{"open-panel-stirling:/usr/share/tesseract-ocr"}},
	"it-tools":      {Container: "open-panel-it-tools", Image: "corentinth/it-tools:latest", Port: "3017:80"},
	"searxng":       {Container: "open-panel-searxng", Image: "searxng/searxng:latest", Port: "3020:8080", Volumes: []string{"open-panel-searxng:/etc/searxng"}},
	"drawio":        {Container: "open-panel-drawio", Image: "jgraph/drawio:latest", Port: "3029:8080"},
	"excalidraw":    {Container: "open-panel-excalidraw", Image: "excalidraw/excalidraw:latest", Port: "3030:80"},
	"mattermost":    {Container: "open-panel-mattermost", Image: "mattermost/mattermost-team-edition:latest", Port: "8065:8065", Volumes: []string{"open-panel-mattermost:/mattermost/data"}},
	"rocketchat":    {Container: "open-panel-rocketchat", Image: "rocket.chat:latest", Port: "3024:3000", Volumes: []string{"open-panel-rocketchat:/app/uploads"}},
	"adminer":       {Container: "open-panel-adminer", Image: "adminer:latest", Port: "3025:8080"},
	"affine":        {Container: "open-panel-affine", Image: "ghcr.io/toeverything/affine-graphql:latest", Port: "3027:3010", Volumes: []string{"open-panel-affine:/root/.affine"}},
	"trilium":       {Container: "open-panel-trilium", Image: "zadam/trilium:latest", Port: "3028:8080", Volumes: []string{"open-panel-trilium:/home/node/trilium-data"}},

	// 多媒体
	"komga":           {Container: "open-panel-komga", Image: "gotson/komga:latest", Port: "8087:25600", Volumes: []string{"open-panel-komga:/config"}},
	"audiobookshelf":  {Container: "open-panel-audiobookshelf", Image: "ghcr.io/advplyr/audiobookshelf:latest", Port: "8094:80", Volumes: []string{"open-panel-abs:/config", "open-panel-abs-meta:/metadata"}},
	"immich":          {Container: "open-panel-immich", Image: "ghcr.io/immich-app/immich-server:release", Port: "3013:2283", Volumes: []string{"open-panel-immich:/usr/src/app/upload"}},

	// 数据库 / 中间件
	"typesense":   {Container: "open-panel-typesense", Image: "typesense/typesense:26.0", Port: "8108:8108", Volumes: []string{"open-panel-typesense:/data"}, Env: []string{"TYPESENSE_API_KEY=openpanel123", "TYPESENSE_DATA_DIR=/data"}},
	"kafka":       {Container: "open-panel-kafka", Image: "apache/kafka:3.9.0", Port: "9092:9092", Env: []string{"KAFKA_NODE_ID=1", "KAFKA_PROCESS_ROLES=broker,controller", "KAFKA_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093", "KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092", "KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER", "KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT", "KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093", "KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1", "KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1", "KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1", "KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=0", "KAFKA_LOG_RETENTION_HOURS=24", "KAFKA_LOG_RETENTION_CHECK_INTERVAL_MS=300000", "KAFKA_HEAP_OPTS=-Xmx512M -Xms256M"}},
	"cloudbeaver": {Container: "open-panel-cloudbeaver", Image: "dbeaver/cloudbeaver:latest", Port: "8088:8978", Volumes: []string{"open-panel-cloudbeaver:/opt/cloudbeaver/workspace"}},
	"jaeger":      {Container: "open-panel-jaeger", Image: "jaegertracing/all-in-one:latest", Port: "16686:16686"},
	"redis-commander": {Container: "open-panel-redis-commander", Image: "rediscommander/redis-commander:latest", Port: "8095:8081", Env: []string{"REDIS_HOSTS=local:host.docker.internal:6379"}},
	"influxdb":    {Container: "open-panel-influxdb", Image: "influxdb:2.7", Port: "8086:8086", Volumes: []string{"open-panel-influxdb:/var/lib/influxdb2"}},
	"clickhouse":  {Container: "open-panel-clickhouse", Image: "clickhouse/clickhouse-server:latest", Port: "8123:8123", Volumes: []string{"open-panel-clickhouse:/var/lib/clickhouse"}},

	// 邮件 / CRM / BI
	"stalwart":  {Container: "open-panel-stalwart", Image: "stalwartlabs/stalwart:latest", Port: "8098:8080", Volumes: []string{"open-panel-stalwart:/opt/stalwart-mail"}},
	"twenty":    {Container: "open-panel-twenty", Image: "twentyhq/twenty:latest", Port: "8103:3000", Volumes: []string{"open-panel-twenty:/app/packages/twenty-server/.local-storage"}},
	"odoo":      {Container: "open-panel-odoo", Image: "odoo:17", Port: "8104:8069", Volumes: []string{"open-panel-odoo:/var/lib/odoo"}, Env: []string{"HOST=host.docker.internal", "USER=odoo", "PASSWORD=odoo"}},
	"jupyter-notebook": {Container: "open-panel-jupyter-nb", Image: "jupyter/scipy-notebook:latest", Port: "8889:8888", Env: []string{"JUPYTER_ENABLE_LAB=yes"}, Volumes: []string{"open-panel-jupyter-nb:/home/jovyan/work"}},
	"umami":     {Container: "open-panel-umami", Image: "ghcr.io/umami-software/umami:postgresql-latest", Port: "3018:3000"},
	"matomo":    {Container: "open-panel-matomo", Image: "matomo:latest", Port: "3019:80", Volumes: []string{"open-panel-matomo:/var/www/html"}},
	"nocodb":    {Container: "open-panel-nocodb", Image: "nocodb/nocodb:latest", Port: "3021:8080", Volumes: []string{"open-panel-nocodb:/usr/app/data"}},

	// 安全
	"zitadel":      {Container: "open-panel-zitadel", Image: "ghcr.io/zitadel/zitadel:latest", Port: "8101:8080"},
	"vaultwarden":  {Container: "open-panel-vaultwarden", Image: "vaultwarden/server:latest", Port: "3012:80", Volumes: []string{"open-panel-vw:/data"}},
	"keycloak":     {Container: "open-panel-keycloak", Image: "quay.io/keycloak/keycloak:latest", Port: "3026:8080", Env: []string{"KEYCLOAK_ADMIN=admin", "KEYCLOAK_ADMIN_PASSWORD=openpanel123"}},

	// 云存储 / 文档
	"paperless-ngx": {Container: "open-panel-paperless", Image: "ghcr.io/paperless-ngx/paperless-ngx:latest", Port: "3014:8000", Volumes: []string{"open-panel-paperless:/usr/src/paperless/data"}},
	"firefly-iii":   {Container: "open-panel-firefly", Image: "fireflyiii/core:latest", Port: "3015:8080", Volumes: []string{"open-panel-firefly:/var/www/html/storage"}},
	"bookstack":     {Container: "open-panel-bookstack", Image: "lscr.io/linuxserver/bookstack:latest", Port: "3022:80", Volumes: []string{"open-panel-bookstack:/config"}},
	"wiki-js":       {Container: "open-panel-wikijs", Image: "requarks/wiki:2", Port: "3023:3000", Volumes: []string{"open-panel-wikijs:/wiki/data"}},

	// 实验性
	"windows": {Container: "open-panel-windows", Image: "dockurr/windows:latest", Port: "8006:8006", Volumes: []string{"open-panel-windows:/storage"}, Env: []string{"RAM_SIZE=4G", "CPU_CORES=2"}},
}
