package appstore

import "fmt"

func tryKafkaInstall(key, version, installPath, dataDir string) (bool, error) {
	if key != "kafka" {
		return false, nil
	}
	_ = version
	_ = installPath
	return true, installKafkaDocker(dataDir)
}

type kafkaImageCandidate struct {
	image string
	env   []string
}

func kafkaApacheEnv() []string {
	return []string{
		"KAFKA_NODE_ID=1",
		"KAFKA_PROCESS_ROLES=broker,controller",
		"KAFKA_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093",
		"KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092",
		"KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER",
		"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT",
		"KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093",
		"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1",
		"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1",
		"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1",
		"KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=0",
		"KAFKA_LOG_RETENTION_HOURS=24",
		"KAFKA_LOG_RETENTION_CHECK_INTERVAL_MS=300000",
		"KAFKA_HEAP_OPTS=-Xmx512M -Xms256M",
	}
}

func kafkaBitnamiEnv() []string {
	return []string{
		"KAFKA_CFG_NODE_ID=0",
		"KAFKA_CFG_PROCESS_ROLES=controller,broker",
		"KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093",
		"KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER",
		"KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=0@localhost:9093",
	}
}

func installKafkaDocker(dataDir string) error {
	if err := ensureDockerEngine(dataDir); err != nil {
		return err
	}
	spec, ok := dockerSpec("kafka")
	if !ok {
		return fmt.Errorf("缺少 Kafka Docker 规格")
	}
	_ = dockerRemove(spec.Container)

	candidates := []kafkaImageCandidate{
		{image: "apache/kafka:3.9.0", env: kafkaApacheEnv()},
		{image: "apache/kafka:latest", env: kafkaApacheEnv()},
		{image: "bitnamilegacy/kafka:3.9", env: kafkaBitnamiEnv()},
		{image: "bitnamilegacy/kafka:latest", env: kafkaBitnamiEnv()},
	}

	var pullErr error
	var chosen kafkaImageCandidate
	for _, c := range candidates {
		logInstallLine(fmt.Sprintf("正在拉取 Kafka 镜像 %s …", c.image))
		if err := runCommand("docker", "pull", c.image); err != nil {
			pullErr = fmt.Errorf("docker pull %s: %w", c.image, err)
			logInstallLine(pullErr.Error())
			continue
		}
		chosen = c
		pullErr = nil
		break
	}
	if pullErr != nil {
		return fmt.Errorf("无法拉取 Kafka 镜像（bitnami/kafka 已从 Docker Hub 下架，请检查网络）: %w", pullErr)
	}

	args := []string{"run", "-d", "--name", spec.Container, "--restart", "unless-stopped"}
	if spec.Port != "" {
		args = append(args, "-p", spec.Port)
	}
	for _, e := range chosen.env {
		args = append(args, "-e", e)
	}
	args = append(args, chosen.image)
	logInstallLine(fmt.Sprintf("正在启动 Kafka 容器（%s）…", chosen.image))
	if err := runCommand("docker", args...); err != nil {
		return fmt.Errorf("docker run %s: %w", spec.Container, err)
	}
	logInstallLine("Kafka 容器已启动")
	return nil
}
