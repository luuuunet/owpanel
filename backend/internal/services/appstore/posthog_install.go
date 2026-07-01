package appstore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/luuuunet/owpanel/internal/secrets"
	"github.com/luuuunet/owpanel/internal/services/settings"
)

const posthogAppKey = "posthog"

func tryPosthogInstall(key, version, installPath, dataDir string) (bool, error) {
	if key != posthogAppKey {
		return false, nil
	}
	_ = version
	_ = installPath
	return true, installPosthogStack(dataDir)
}

func tryPosthogUninstall(key, dataDir string) (bool, error) {
	if key != posthogAppKey {
		return false, nil
	}
	return true, uninstallPosthogStack(dataDir)
}

func tryPosthogServiceAction(key, action, dataDir string) (bool, error) {
	if key != posthogAppKey {
		return false, nil
	}
	dir := posthogAppDir(dataDir)
	cf := filepath.Join(dir, "docker-compose.yml")
	if _, err := os.Stat(cf); err != nil {
		return true, fmt.Errorf("PostHog 尚未安装")
	}
	switch action {
	case "start":
		return true, runDockerComposeInDir(dir, "up", "-d")
	case "stop":
		return true, runDockerComposeInDir(dir, "stop")
	case "restart":
		_ = runDockerComposeInDir(dir, "stop")
		return true, runDockerComposeInDir(dir, "up", "-d")
	default:
		return true, nil
	}
}

func posthogAppDir(dataDir string) string {
	return settings.DockerAppPath(dataDir, posthogAppKey)
}

func PosthogInstalled(dataDir string) bool {
	cf := filepath.Join(posthogAppDir(dataDir), "docker-compose.yml")
	_, err := os.Stat(cf)
	return err == nil
}

func PosthogComposeStatus(dataDir string) string {
	dir := posthogAppDir(dataDir)
	cf := filepath.Join(dir, "docker-compose.yml")
	if _, err := os.Stat(cf); err != nil {
		return "stopped"
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return "unknown"
	}
	if dockerComposePSRunning(dir) {
		return "running"
	}
	return "stopped"
}

func installPosthogStack(dataDir string) error {
	if err := ensureDockerEngine(dataDir); err != nil {
		return err
	}
	dir := posthogAppDir(dataDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create app dir: %w", err)
	}

	pgPass, _ := secrets.GeneratePassword(24)
	chPass, _ := secrets.GeneratePassword(24)
	secretKey, _ := secrets.GeneratePassword(48)
	minioUser := "posthog_minio"
	minioPass, _ := secrets.GeneratePassword(24)
	if pgPass == "" {
		pgPass = "posthog_pg_changeme"
	}
	if chPass == "" {
		chPass = pgPass
	}
	if secretKey == "" {
		secretKey = pgPass + pgPass
	}
	if minioPass == "" {
		minioPass = pgPass
	}

	siteURL := "http://127.0.0.1:8020"

	composePath := filepath.Join(dir, "docker-compose.yml")
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(composePath, []byte(posthogComposeYAML()), 0644); err != nil {
		return fmt.Errorf("write docker-compose.yml: %w", err)
	}
	if err := os.WriteFile(envPath, []byte(posthogEnvFile(siteURL, secretKey, pgPass, chPass, minioUser, minioPass)), 0600); err != nil {
		return fmt.Errorf("write .env: %w", err)
	}
	logInstallLine(fmt.Sprintf("PostHog 配置已写入 %s", dir))

	_ = runDockerComposeInDir(dir, "down", "--remove-orphans")
	if err := runDockerComposeInDir(dir, "pull"); err != nil {
		logInstallLine("docker compose pull 警告: " + err.Error())
	}

	logInstallLine("启动 PostHog 基础服务（PostgreSQL / Redis / Zookeeper）…")
	if err := runDockerComposeInDir(dir, "up", "-d", "db", "redis", "zookeeper"); err != nil {
		return fmt.Errorf("start base services: %w", err)
	}
	time.Sleep(15 * time.Second)

	logInstallLine("启动 Kafka / ClickHouse / MinIO…")
	if err := runDockerComposeInDir(dir, "up", "-d", "kafka", "clickhouse", "object_storage"); err != nil {
		return fmt.Errorf("start data services: %w", err)
	}
	time.Sleep(30 * time.Second)

	logInstallLine("创建 MinIO bucket…")
	_ = runDockerComposeInDir(dir, "up", "createbuckets")

	logInstallLine("运行 PostHog 数据库迁移…")
	if err := runDockerComposeInDir(dir, "run", "--rm", "web", "./bin/migrate"); err != nil {
		logInstallLine("migrate 警告: " + err.Error())
	}

	logInstallLine("启动 PostHog 应用栈…")
	if err := runDockerComposeInDir(dir, "up", "-d"); err != nil {
		return fmt.Errorf("docker compose up: %w", err)
	}
	logInstallLine("PostHog 已启动（仪表盘 :8020，建议 8GB+ RAM）")
	return nil
}

func uninstallPosthogStack(dataDir string) error {
	dir := posthogAppDir(dataDir)
	cf := filepath.Join(dir, "docker-compose.yml")
	if _, err := os.Stat(cf); err == nil {
		if _, lookErr := exec.LookPath("docker"); lookErr == nil {
			_ = runDockerComposeInDir(dir, "down", "-v")
		}
	}
	return os.RemoveAll(dir)
}

func posthogEnvFile(siteURL, secretKey, pgPass, chPass, minioUser, minioPass string) string {
	return fmt.Sprintf(`SITE_URL=%s
SECRET_KEY=%s
POSTGRES_PASSWORD=%s
CLICKHOUSE_PASSWORD=%s
MINIO_ROOT_USER=%s
MINIO_ROOT_PASSWORD=%s
DISABLE_SECURE_SSL_REDIRECT=true
IS_BEHIND_PROXY=true
`, siteURL, secretKey, pgPass, chPass, minioUser, minioPass)
}

func posthogComposeYAML() string {
	return `x-posthog-env: &posthog-env
  SECRET_KEY: ${SECRET_KEY}
  DATABASE_URL: postgres://posthog:${POSTGRES_PASSWORD}@db:5432/posthog
  REDIS_URL: redis://redis:6379/
  CLICKHOUSE_HOST: clickhouse
  CLICKHOUSE_DATABASE: posthog
  CLICKHOUSE_USER: default
  CLICKHOUSE_PASSWORD: ${CLICKHOUSE_PASSWORD}
  KAFKA_HOSTS: kafka:9092
  OBJECT_STORAGE_ENABLED: "true"
  OBJECT_STORAGE_ENDPOINT: http://object_storage:9000
  OBJECT_STORAGE_ACCESS_KEY_ID: ${MINIO_ROOT_USER}
  OBJECT_STORAGE_SECRET_ACCESS_KEY: ${MINIO_ROOT_PASSWORD}
  OBJECT_STORAGE_BUCKET: posthog
  SITE_URL: ${SITE_URL}
  DISABLE_SECURE_SSL_REDIRECT: "true"
  IS_BEHIND_PROXY: "true"

services:
  db:
    container_name: owpanel-posthog-db
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      POSTGRES_DB: posthog
      POSTGRES_USER: posthog
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - posthog-pg:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U posthog -d posthog"]
      interval: 10s
      timeout: 5s
      retries: 10

  redis:
    container_name: owpanel-posthog-redis
    image: redis:7-alpine
    restart: unless-stopped
    command: redis-server --maxmemory 256mb --maxmemory-policy allkeys-lru
    volumes:
      - posthog-redis:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  zookeeper:
    container_name: owpanel-posthog-zk
    image: zookeeper:3.7.0
    restart: unless-stopped
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    volumes:
      - posthog-zk-data:/data
      - posthog-zk-logs:/datalog
    healthcheck:
      test: ["CMD-SHELL", "echo ruok | nc localhost 2181 | grep imok"]
      interval: 15s
      timeout: 10s
      retries: 10
      start_period: 30s

  kafka:
    container_name: owpanel-posthog-kafka
    image: bitnami/kafka:2.8.1-debian-11-r72
    restart: unless-stopped
    depends_on:
      zookeeper:
        condition: service_healthy
    environment:
      KAFKA_CFG_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_CFG_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_CFG_LISTENERS: PLAINTEXT://:9092
      ALLOW_PLAINTEXT_LISTENER: "yes"
      KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE: "true"
    volumes:
      - posthog-kafka:/bitnami/kafka
    healthcheck:
      test: ["CMD-SHELL", "kafka-topics.sh --bootstrap-server localhost:9092 --list 2>/dev/null || exit 1"]
      interval: 30s
      timeout: 15s
      retries: 10
      start_period: 60s

  clickhouse:
    container_name: owpanel-posthog-ch
    image: clickhouse/clickhouse-server:23.8.3.24
    restart: unless-stopped
    depends_on:
      zookeeper:
        condition: service_healthy
    environment:
      CLICKHOUSE_PASSWORD: ${CLICKHOUSE_PASSWORD}
    volumes:
      - posthog-ch:/var/lib/clickhouse
    ulimits:
      nofile:
        soft: 262144
        hard: 262144
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8123/ping"]
      interval: 15s
      timeout: 10s
      retries: 10
      start_period: 60s

  object_storage:
    container_name: owpanel-posthog-minio
    image: minio/minio:RELEASE.2024-08-17T01-24-54Z
    restart: unless-stopped
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    command: server /data --console-address ":9001"
    volumes:
      - posthog-minio:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 15s
      timeout: 10s
      retries: 5

  createbuckets:
    container_name: owpanel-posthog-minio-init
    image: minio/mc:RELEASE.2024-08-20T22-49-07Z
    depends_on:
      object_storage:
        condition: service_healthy
    env_file:
      - .env
    entrypoint: >
      /bin/sh -c "
      mc alias set minio http://object_storage:9000 $${MINIO_ROOT_USER} $${MINIO_ROOT_PASSWORD};
      mc mb --ignore-existing minio/posthog;
      exit 0;
      "

  web:
    container_name: owpanel-posthog
    image: ghcr.io/posthog/posthog:release-latest
    restart: unless-stopped
    command: ./bin/docker-server
    ports:
      - "8020:8000"
    environment:
      <<: *posthog-env
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_healthy
      clickhouse:
        condition: service_healthy
      kafka:
        condition: service_healthy
      object_storage:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8000/_health/"]
      interval: 15s
      timeout: 10s
      retries: 10
      start_period: 120s

  worker:
    container_name: owpanel-posthog-worker
    image: ghcr.io/posthog/posthog:release-latest
    restart: unless-stopped
    command: ./bin/docker-worker-celery --without-gossip --without-mingle --without-heartbeat
    environment:
      <<: *posthog-env
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_healthy
      clickhouse:
        condition: service_healthy

  beat:
    container_name: owpanel-posthog-beat
    image: ghcr.io/posthog/posthog:release-latest
    restart: unless-stopped
    command: ./bin/docker-worker-beat --without-gossip --without-mingle --without-heartbeat
    environment:
      <<: *posthog-env
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_healthy

  plugins:
    container_name: owpanel-posthog-plugins
    image: ghcr.io/posthog/posthog:release-latest
    restart: unless-stopped
    command: ./bin/plugin-server --no-restart-loop
    environment:
      <<: *posthog-env
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_healthy
      clickhouse:
        condition: service_healthy
      kafka:
        condition: service_healthy

volumes:
  posthog-pg:
  posthog-redis:
  posthog-zk-data:
  posthog-zk-logs:
  posthog-kafka:
  posthog-ch:
  posthog-minio:
`
}
