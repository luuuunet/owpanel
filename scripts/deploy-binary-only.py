"""Deploy owpanel binary only (fast hotfix)."""
import paramiko

HOST = "198.199.120.139"
USER = "root"
PASSWORD = "Wuyfieng0Wuyifeng"
BINARY = r"C:\Users\Administrator\Projects\open-panel\dist-fix-owpanel"


def run(ssh, cmd, timeout=120):
    print(">>>", cmd[:160])
    _, stdout, stderr = ssh.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", "replace")
    err = stderr.read().decode("utf-8", "replace")
    if out.strip():
        print(out[:2000])
    if err.strip():
        print("ERR:", err[:2000])


def main():
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(HOST, username=USER, password=PASSWORD, timeout=20)
    sftp = ssh.open_sftp()
    try:
        sftp.put(BINARY, "/opt/owpanel/owpanel.new")
        run(
            ssh,
            "chmod +x /opt/owpanel/owpanel.new && "
            "cp /opt/owpanel/owpanel /opt/owpanel/owpanel.bak 2>/dev/null; "
            "mv /opt/owpanel/owpanel.new /opt/owpanel/owpanel && "
            "systemctl restart owpanel && sleep 3 && systemctl is-active owpanel",
        )
    finally:
        sftp.close()
        ssh.close()


if __name__ == "__main__":
    main()
