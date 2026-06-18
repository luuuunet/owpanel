package appstore

// dockerAppSpec defines a one-click Docker deployment.
type dockerAppSpec struct {
	Container string
	Image     string
	Port      string // host:container
	Env       []string
	Volumes   []string // host:container
}

// dockerAppSpecs — 内置 Docker 一键部署规格（需先安装 Docker）
var dockerAppSpecs = map[string]dockerAppSpec{
	// AI (shared with installer_ai)
	"open-webui":  {Container: "open-panel-open-webui", Image: "ghcr.io/open-webui/open-webui:main", Port: "8080:8080", Env: []string{"OLLAMA_BASE_URL=http://host.docker.internal:11434"}},
	"localai":     {Container: "open-panel-localai", Image: "localai/localai:latest", Port: "8090:8080"},
	"dify":        {Container: "open-panel-dify", Image: "langgenius/dify-web:latest", Port: "8091:3000"},
	"anythingllm": {Container: "open-panel-anythingllm", Image: "mintplexlabs/anythingllm:latest", Port: "3001:3001"},
	"fastgpt":     {Container: "open-panel-fastgpt", Image: "ghcr.io/labring/fastgpt:latest", Port: "3002:3000"},
	"comfyui":     {Container: "open-panel-comfyui", Image: "yanwk/comfyui-boot:cu124-slim", Port: "8188:8188"},
	"sd-webui":    {Container: "open-panel-sd-webui", Image: "continuumio/miniconda3:latest", Port: "7860:7860"},

	// 建站 / CMS
	"halo":              {Container: "open-panel-halo", Image: "halohub/halo:2.20", Port: "8090:8090"},
	"typecho":           {Container: "open-panel-typecho", Image: "joyqi/typecho:nightly-php8.2-apache", Port: "8080:80"},
	"wordpress-app":     {Container: "open-panel-wordpress", Image: "wordpress:6.7-php8.2-apache", Port: "8081:80", Env: []string{"WORDPRESS_DB_HOST=host.docker.internal"}},
	"ghost":             {Container: "open-panel-ghost", Image: "ghost:5-alpine", Port: "2368:2368", Env: []string{"url=http://localhost:2368"}},
	"outline":           {Container: "open-panel-outline", Image: "outlinewiki/outline:latest", Port: "3005:3000"},
	"memos":             {Container: "open-panel-memos", Image: "neosmemo/memos:stable", Port: "5230:5230"},
	"lsky-pro":          {Container: "open-panel-lsky", Image: "ddsderek/lsky-pro:latest", Port: "8099:80"},
	"flarum":            {Container: "open-panel-flarum", Image: "monologg/flarum-docker:latest", Port: "8889:8888"},

	// DevOps
	"portainer":    {Container: "open-panel-portainer", Image: "portainer/portainer-ce:latest", Port: "9000:9000", Volumes: []string{"/var/run/docker.sock:/var/run/docker.sock", "open-panel-portainer-data:/data"}},
	"gitea":        {Container: "open-panel-gitea", Image: "gitea/gitea:latest", Port: "3000:3000", Volumes: []string{"open-panel-gitea-data:/data"}},
	"jenkins":      {Container: "open-panel-jenkins", Image: "jenkins/jenkins:lts-jdk17", Port: "8082:8080", Volumes: []string{"open-panel-jenkins:/var/jenkins_home"}},
	"gitlab":       {Container: "open-panel-gitlab", Image: "gitlab/gitlab-ce:latest", Port: "8929:8929", Volumes: []string{"open-panel-gitlab-config:/etc/gitlab", "open-panel-gitlab-logs:/var/log/gitlab", "open-panel-gitlab-data:/var/opt/gitlab"}},
	"uptime-kuma":  {Container: "open-panel-uptime-kuma", Image: "louislam/uptime-kuma:1", Port: "3004:3001", Volumes: []string{"open-panel-uptime-kuma:/app/data"}},
	"netdata":      {Container: "open-panel-netdata", Image: "netdata/netdata:stable", Port: "19999:19999", Volumes: []string{"/proc:/host/proc:ro", "/sys:/host/sys:ro", "/var/run/docker.sock:/var/run/docker.sock:ro"}},
	"sonarqube":    {Container: "open-panel-sonarqube", Image: "sonarqube:lts-community", Port: "9002:9000", Volumes: []string{"open-panel-sonarqube-data:/opt/sonarqube/data"}},
	"frps":         {Container: "open-panel-frps", Image: "snowdreamtech/frps:latest", Port: "7500:7500", Volumes: []string{"open-panel-frps:/etc/frp"}},

	// 中间件
	"minio":           {Container: "open-panel-minio", Image: "minio/minio:latest", Port: "9001:9001", Env: []string{"MINIO_ROOT_USER=admin", "MINIO_ROOT_PASSWORD=openpanel123"}, Volumes: []string{"open-panel-minio:/data"}},
	"rabbitmq":        {Container: "open-panel-rabbitmq", Image: "rabbitmq:3-management-alpine", Port: "15672:15672", Env: []string{"RABBITMQ_DEFAULT_USER=admin", "RABBITMQ_DEFAULT_PASS=openpanel123"}},
	"elasticsearch":   {Container: "open-panel-elasticsearch", Image: "elasticsearch:8.11.0", Port: "9200:9200", Env: []string{"discovery.type=single-node", "xpack.security.enabled=false", "ES_JAVA_OPTS=-Xms512m -Xmx512m"}, Volumes: []string{"open-panel-es-data:/usr/share/elasticsearch/data"}},
	"nacos":           {Container: "open-panel-nacos", Image: "nacos/nacos-server:v2.3.2", Port: "8848:8848", Env: []string{"MODE=standalone"}},
	"etcd":            {Container: "open-panel-etcd", Image: "quay.io/coreos/etcd:v3.5.16", Port: "2379:2379", Env: []string{"ETCD_ADVERTISE_CLIENT_URLS=http://127.0.0.1:2379", "ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379"}},
	"traefik":         {Container: "open-panel-traefik", Image: "traefik:v3.0", Port: "8083:8080", Volumes: []string{"/var/run/docker.sock:/var/run/docker.sock:ro"}},
	"caddy":           {Container: "open-panel-caddy", Image: "caddy:2-alpine", Port: "2015:2015", Volumes: []string{"open-panel-caddy-data:/data", "open-panel-caddy-config:/config"}},

	// 数据库工具
	"mongo-express": {Container: "open-panel-mongo-express", Image: "mongo-express:latest", Port: "8082:8081", Env: []string{"ME_CONFIG_MONGODB_URL=mongodb://host.docker.internal:27017"}},
	"redis-insight": {Container: "open-panel-redis-insight", Image: "redis/redisinsight:latest", Port: "5540:5540"},
	"pgadmin":       {Container: "open-panel-pgadmin", Image: "dpage/pgadmin4:latest", Port: "5050:80", Env: []string{"PGADMIN_DEFAULT_EMAIL=admin@openpanel.local", "PGADMIN_DEFAULT_PASSWORD=openpanel123"}},

	// BI
	"kibana":    {Container: "open-panel-kibana", Image: "kibana:8.11.0", Port: "5601:5601", Env: []string{"ELASTICSEARCH_HOSTS=http://host.docker.internal:9200"}},
	"prometheus": {Container: "open-panel-prometheus", Image: "prom/prometheus:latest", Port: "9090:9090", Volumes: []string{"open-panel-prometheus:/prometheus"}},
	"grafana":   {Container: "open-panel-grafana", Image: "grafana/grafana:latest", Port: "3003:3000", Volumes: []string{"open-panel-grafana:/var/lib/grafana"}},
	"metabase":  {Container: "open-panel-metabase", Image: "metabase/metabase:latest", Port: "3007:3000"},
	"superset":  {Container: "open-panel-superset", Image: "apache/superset:latest", Port: "8089:8088"},

	// 开发工具
	"code-server": {Container: "open-panel-code-server", Image: "codercom/code-server:latest", Port: "8088:8080", Env: []string{"PASSWORD=openpanel123"}, Volumes: []string{"open-panel-code-server:/home/coder"}},
	"hoppscotch":  {Container: "open-panel-hoppscotch", Image: "hoppscotch/hoppscotch:latest", Port: "3006:3000"},

	// 多媒体 / 生活
	"home-assistant": {Container: "open-panel-homeassistant", Image: "ghcr.io/home-assistant/home-assistant:stable", Port: "8123:8123", Volumes: []string{"open-panel-ha:/config"}},
	"qbittorrent":    {Container: "open-panel-qbittorrent", Image: "lscr.io/linuxserver/qbittorrent:latest", Port: "8085:8080", Volumes: []string{"open-panel-qbit-config:/config", "open-panel-qbit-downloads:/downloads"}},
	"jellyfin":       {Container: "open-panel-jellyfin", Image: "jellyfin/jellyfin:latest", Port: "8096:8096", Volumes: []string{"open-panel-jellyfin-config:/config", "open-panel-jellyfin-cache:/cache"}},
	"navidrome":      {Container: "open-panel-navidrome", Image: "deluan/navidrome:latest", Port: "4533:4533", Volumes: []string{"open-panel-navidrome:/data"}},
	"emby":           {Container: "open-panel-emby", Image: "emby/embyserver:latest", Port: "8097:8096", Volumes: []string{"open-panel-emby:/config"}},
	"nextcloud":      {Container: "open-panel-nextcloud", Image: "nextcloud:latest", Port: "8087:80", Volumes: []string{"open-panel-nextcloud:/var/www/html"}},
	"syncthing":      {Container: "open-panel-syncthing", Image: "syncthing/syncthing:latest", Port: "8384:8384", Volumes: []string{"open-panel-syncthing:/var/syncthing"}},
	"alist":          {Container: "open-panel-alist", Image: "xhofe/alist:latest", Port: "5244:5244", Volumes: []string{"open-panel-alist:/opt/alist/data"}},
	"filebrowser":    {Container: "open-panel-filebrowser", Image: "filebrowser/filebrowser:latest", Port: "8086:80", Volumes: []string{"open-panel-filebrowser:/srv"}},

	// 安全
	"casdoor": {Container: "open-panel-casdoor", Image: "casbin/casdoor:latest", Port: "8000:8000"},

	// 图形处理
	"photoprism": {Container: "open-panel-photoprism", Image: "photoprism/photoprism:latest", Port: "2342:2342", Env: []string{"PHOTOPRISM_ADMIN_PASSWORD=openpanel123", "PHOTOPRISM_DATABASE_DRIVER=sqlite", "PHOTOPRISM_ORIGINALS_LIMIT=5000", "PHOTOPRISM_HTTP_COMPRESSION=gzip"}, Volumes: []string{"open-panel-photoprism-storage:/photoprism/storage", "open-panel-photoprism-originals:/photoprism/originals"}},
	"pigallery2": {Container: "open-panel-pigallery2", Image: "bpatrik/pigallery2:latest", Port: "3031:80", Volumes: []string{"open-panel-pigallery2:/app/data/config", "open-panel-pigallery2-db:/app/data/db", "open-panel-pigallery2-images:/app/data/images", "open-panel-pigallery2-tmp:/app/data/tmp"}},
	"imgproxy":   {Container: "open-panel-imgproxy", Image: "imgproxy/imgproxy:latest", Port: "3032:8080", Env: []string{"IMGPROXY_BIND=:8080", "IMGPROXY_USE_ETAG=true"}},
	"imagor":     {Container: "open-panel-imagor", Image: "shumc/imagor:latest", Port: "3033:8000", Env: []string{"IMAGOR_SECRET=openpanel123", "IMAGOR_AUTO_WEBP=true"}},
	"thumbor":    {Container: "open-panel-thumbor", Image: "beamerblbo/thumbor:latest", Port: "3037:8000", Env: []string{"THUMBOR_SECURITY_KEY=openpanel123"}},

	// 视频处理
	"tdarr":      {Container: "open-panel-tdarr", Image: "ghcr.io/haveagitgat/tdarr:latest", Port: "8265:8265", Env: []string{"serverIP=0.0.0.0", "serverPort=8266", "webUIPort=8265"}, Volumes: []string{"open-panel-tdarr-server:/app/server", "open-panel-tdarr-configs:/app/configs", "open-panel-tdarr-logs:/app/logs"}},
	"unmanic":    {Container: "open-panel-unmanic", Image: "ghcr.io/unmanic/unmanic:latest", Port: "3038:37488", Volumes: []string{"open-panel-unmanic-config:/config", "open-panel-unmanic-temp:/tmp/unmanic"}},
	"handbrake":  {Container: "open-panel-handbrake", Image: "lscr.io/linuxserver/handbrake:latest", Port: "5800:5800", Volumes: []string{"open-panel-handbrake-config:/config"}},
	"fileflows":  {Container: "open-panel-fileflows", Image: "revenz/fileflows:latest", Port: "3040:5000", Volumes: []string{"open-panel-fileflows:/app/Data"}},
}

func dockerSpec(key string) (dockerAppSpec, bool) {
	spec, ok := dockerAppSpecs[key]
	return spec, ok
}
