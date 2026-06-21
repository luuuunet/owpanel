package appstore

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luuuunet/owpanel/internal/secrets"
	"github.com/luuuunet/owpanel/internal/services/settings"
)

var dataPlatformComposeKeys = map[string]bool{
	"milvus": true, "weaviate": true, "victoria-metrics": true, "ceph": true, "vllm": true,
}

func tryDataPlatformInstall(key, version, installPath, dataDir string) (bool, error) {
	if !dataPlatformComposeKeys[key] {
		return false, nil
	}
	_ = version
	_ = installPath
	return true, installDataPlatformCompose(key, dataDir)
}

func tryDataPlatformUninstall(key, dataDir string) (bool, error) {
	if !dataPlatformComposeKeys[key] {
		return false, nil
	}
	return true, uninstallDataPlatformCompose(key, dataDir)
}

func tryDataPlatformServiceAction(key, action, dataDir string) (bool, error) {
	if !dataPlatformComposeKeys[key] {
		return false, nil
	}
	dir := dataPlatformAppDir(dataDir, key)
	cf := filepath.Join(dir, "docker-compose.yml")
	if _, err := os.Stat(cf); err != nil {
		return true, fmt.Errorf("%s 尚未安装", key)
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

func tryDataPlatformStatus(key, dataDir string) (bool, string) {
	if !dataPlatformComposeKeys[key] {
		return false, ""
	}
	if !DataPlatformComposeInstalled(dataDir, key) {
		return false, ""
	}
	if DataPlatformComposeRunning(dataDir, key) {
		return true, "running"
	}
	return true, "stopped"
}

func dataPlatformAppDir(dataDir, key string) string {
	return settings.DockerAppPath(dataDir, key)
}

func DataPlatformComposeInstalled(dataDir, key string) bool {
	cf := filepath.Join(dataPlatformAppDir(dataDir, key), "docker-compose.yml")
	_, err := os.Stat(cf)
	return err == nil
}

func DataPlatformComposeRunning(dataDir, key string) bool {
	dir := dataPlatformAppDir(dataDir, key)
	if _, err := os.Stat(filepath.Join(dir, "docker-compose.yml")); err != nil {
		return false
	}
	return dockerComposePSRunning(dir)
}

func installDataPlatformCompose(key, dataDir string) error {
	if err := ensureDockerEngine(dataDir); err != nil {
		return err
	}
	dir := dataPlatformAppDir(dataDir, key)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	compose, env, err := dataPlatformComposeFiles(key, dataDir)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644); err != nil {
		return err
	}
	if env != "" {
		if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0600); err != nil {
			return err
		}
	}
	logInstallLine(fmt.Sprintf("%s compose 已写入 %s", key, dir))
	_ = runDockerComposeInDir(dir, "down", "--remove-orphans")
	if err := runDockerComposeInDir(dir, "pull"); err != nil {
		logInstallLine("compose pull 警告: " + err.Error())
	}
	if err := runDockerComposeInDir(dir, "up", "-d"); err != nil {
		return fmt.Errorf("compose up: %w", err)
	}
	logInstallLine(key + " 已启动")
	return nil
}

func uninstallDataPlatformCompose(key, dataDir string) error {
	dir := dataPlatformAppDir(dataDir, key)
	cf := filepath.Join(dir, "docker-compose.yml")
	if _, err := os.Stat(cf); err == nil {
		_ = runDockerComposeInDir(dir, "down", "-v")
	}
	return os.RemoveAll(dir)
}

func dataPlatformComposeFiles(key, dataDir string) (compose, env string, err error) {
	switch key {
	case "milvus":
		return milvusComposeYAML(), "", nil
	case "weaviate":
		return weaviateComposeYAML(), "", nil
	case "victoria-metrics":
		return victoriaMetricsComposeYAML(), "", nil
	case "ceph":
		pass, e := secrets.GeneratePassword(20)
		if e != nil {
			pass = "openpanel123"
		}
		env = fmt.Sprintf("CEPH_DEMO_ACCESS_KEY=admin\nCEPH_DEMO_SECRET_KEY=%s\n", pass)
		dir := filepath.Join(dataDir, "docker-secrets")
		_ = os.MkdirAll(dir, 0700)
		_ = os.WriteFile(filepath.Join(dir, "ceph.env"), []byte(env), 0600)
		return cephComposeYAML(), env, nil
	case "vllm":
		return vllmComposeYAML(), "", nil
	default:
		return "", "", fmt.Errorf("unknown dataplatform app: %s", key)
	}
}

func milvusComposeYAML() string {
	return `services:
  etcd:
    container_name: owpanel-milvus-etcd
    image: quay.io/coreos/etcd:v3.5.5
    restart: unless-stopped
    environment:
      ETCD_AUTO_COMPACTION_MODE: revision
      ETCD_AUTO_COMPACTION_RETENTION: "1000"
      ETCD_QUOTA_BACKEND_BYTES: "4294967296"
      ETCD_SNAPSHOT_COUNT: "50000"
    volumes:
      - milvus-etcd:/etcd
    command: etcd -advertise-client-urls=http://127.0.0.1:2379 -listen-client-urls http://0.0.0.0:2379 --data-dir /etcd

  minio:
    container_name: owpanel-milvus-minio
    image: minio/minio:RELEASE.2023-03-15T23-07-09Z
    restart: unless-stopped
    environment:
      MINIO_ACCESS_KEY: minioadmin
      MINIO_SECRET_KEY: minioadmin
    volumes:
      - milvus-minio:/minio_data
    command: minio server /minio_data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  standalone:
    container_name: owpanel-milvus
    image: milvusdb/milvus:v2.4.4
    restart: unless-stopped
    command: ["milvus", "run", "standalone"]
    security_opt:
      - seccomp:unconfined
    environment:
      ETCD_ENDPOINTS: etcd:2379
      MINIO_ADDRESS: minio:9000
    volumes:
      - milvus-data:/var/lib/milvus
    ports:
      - "19530:19530"
      - "9091:9091"
    depends_on:
      - etcd
      - minio

volumes:
  milvus-etcd:
  milvus-minio:
  milvus-data:
`
}

func weaviateComposeYAML() string {
	return `services:
  weaviate:
    container_name: owpanel-weaviate
    image: cr.weaviate.io/semitechnologies/weaviate:1.27.0
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "50051:50051"
    volumes:
      - weaviate-data:/var/lib/weaviate
    environment:
      QUERY_DEFAULTS_LIMIT: 25
      AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED: "true"
      PERSISTENCE_DATA_PATH: /var/lib/weaviate
      DEFAULT_VECTORIZER_MODULE: none
      CLUSTER_HOSTNAME: node1

volumes:
  weaviate-data:
`
}

func victoriaMetricsComposeYAML() string {
	return `services:
  victoria-metrics:
    container_name: owpanel-victoria-metrics
    image: victoriametrics/victoria-metrics:v1.103.0
    restart: unless-stopped
    ports:
      - "8428:8428"
    volumes:
      - vm-data:/victoria-metrics-data
    command:
      - "--storageDataPath=/victoria-metrics-data"
      - "--httpListenAddr=:8428"
      - "--retentionPeriod=12"

volumes:
  vm-data:
`
}

func cephComposeYAML() string {
	return `services:
  ceph-demo:
    container_name: owpanel-ceph-rgw
    image: quay.io/ceph/demo:latest
    restart: unless-stopped
    privileged: true
    ports:
      - "7480:7480"
      - "8443:8443"
    environment:
      MON_IP: 127.0.0.1
      CEPH_PUBLIC_NETWORK: 0.0.0.0/0
      RGW_NAME: default
    volumes:
      - ceph-demo:/var/lib/ceph

volumes:
  ceph-demo:
`
}

func vllmComposeYAML() string {
	return `services:
  vllm:
    container_name: owpanel-vllm
    image: vllm/vllm-openai:latest
    restart: unless-stopped
    ports:
      - "8000:8000"
    volumes:
      - vllm-cache:/root/.cache
    ipc: host
    command: ["--model", "Qwen/Qwen2.5-0.5B-Instruct", "--host", "0.0.0.0", "--port", "8000"]

volumes:
  vllm-cache:
`
}
