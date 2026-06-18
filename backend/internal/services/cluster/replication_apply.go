package cluster

import (
	"fmt"
	"strings"
)

// ApplyReplicationViaSSH runs replication setup on master and slave when SSH is configured.
func (s *Service) ApplyReplicationViaSSH(master, slave *FlowNode) (string, error) {
	if master == nil || slave == nil {
		return "", fmt.Errorf("主从节点不能为空")
	}
	masterNode, err := s.GetNode(master.RefID)
	if err != nil {
		return "", fmt.Errorf("主库节点不存在")
	}
	slaveNode, err := s.GetNode(slave.RefID)
	if err != nil {
		return "", fmt.Errorf("从库节点不存在")
	}

	replUser := strCfg(master.Config, "repl_user", "repl")
	replPass := strCfg(master.Config, "repl_password", "")
	if replPass == "" {
		replPass = "Repl_" + randomToken(4)
		if master.Config == nil {
			master.Config = map[string]interface{}{}
		}
		master.Config["repl_password"] = replPass
	}
	dbName := strCfg(master.Config, "db_name", "app_db")
	mHost := s.nodeHost(master)

	var log strings.Builder

	// Master side
	if nodeHasSSH(masterNode) {
		mSQL := fmt.Sprintf(`set -e
mysql -e "CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s';" 2>/dev/null || mysql -uroot -e "CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s';"
mysql -e "GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO '%s'@'%%'; FLUSH PRIVILEGES;" 2>/dev/null || mysql -uroot -e "GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO '%s'@'%%'; FLUSH PRIVILEGES;"
mysql -e "CREATE DATABASE IF NOT EXISTS %s;" 2>/dev/null || mysql -uroot -e "CREATE DATABASE IF NOT EXISTS %s;"
mysql -e "SHOW MASTER STATUS\\G" 2>/dev/null || mysql -uroot -e "SHOW MASTER STATUS\\G"
echo master_done`,
			replUser, replPass, replUser, replPass,
			replUser, replUser, dbName, dbName)
		client, err := s.sshClient(masterNode)
		if err != nil {
			log.WriteString(fmt.Sprintf("主库 SSH 失败: %s\n", err.Error()))
		} else {
			out, runErr := sshRun(client, mSQL)
			log.WriteString("=== 主库 ===\n" + out + "\n")
			client.Close()
			if runErr != nil {
				log.WriteString(fmt.Sprintf("主库执行警告: %s\n", runErr.Error()))
			}
		}
	} else {
		log.WriteString(fmt.Sprintf("主库 %s 无 SSH，跳过远程执行（已生成 SQL 脚本）\n", masterNode.Name))
	}

	// Slave side
	if nodeHasSSH(slaveNode) {
		sSQL := fmt.Sprintf(`set -e
mysql -e "STOP SLAVE;" 2>/dev/null || mysql -uroot -e "STOP SLAVE;" || true
mysql -e "CHANGE MASTER TO MASTER_HOST='%s', MASTER_USER='%s', MASTER_PASSWORD='%s', MASTER_AUTO_POSITION=1;" 2>/dev/null || mysql -uroot -e "CHANGE MASTER TO MASTER_HOST='%s', MASTER_USER='%s', MASTER_PASSWORD='%s', MASTER_AUTO_POSITION=1;"
mysql -e "START SLAVE;" 2>/dev/null || mysql -uroot -e "START SLAVE;"
mysql -e "SHOW SLAVE STATUS\\G" 2>/dev/null || mysql -uroot -e "SHOW SLAVE STATUS\\G"
echo slave_done`,
			mHost, replUser, replPass, mHost, replUser, replPass)
		client, err := s.sshClient(slaveNode)
		if err != nil {
			log.WriteString(fmt.Sprintf("从库 SSH 失败: %s\n", err.Error()))
			return log.String(), err
		}
		out, runErr := sshRun(client, sSQL)
		log.WriteString("=== 从库 ===\n" + out + "\n")
		client.Close()
		if runErr != nil {
			return log.String(), fmt.Errorf("从库复制配置失败: %w", runErr)
		}
		log.WriteString(fmt.Sprintf("主从复制已应用到 %s → %s\n", masterNode.Name, slaveNode.Name))
	} else {
		log.WriteString(fmt.Sprintf("从库 %s 无 SSH，请手动执行 data/cluster/replication/ 下的 SQL\n", slaveNode.Name))
	}

	_ = s.ensureReplicationDBRecords(master, slave)
	return log.String(), nil
}
