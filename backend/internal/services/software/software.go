package software

type Item struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
	Type    string `json:"type"`
}

type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) List() []Item {
	return []Item{
		{Name: "OpenResty", Version: "1.25.3", Status: "running", Type: "Web服务器"},
		{Name: "MySQL", Version: "8.0.36", Status: "running", Type: "数据库"},
		{Name: "PHP", Version: "8.3.6", Status: "running", Type: "运行环境"},
		{Name: "Redis", Version: "7.2.4", Status: "stopped", Type: "数据库"},
		{Name: "Docker", Version: "24.0.7", Status: "running", Type: "容器"},
		{Name: "Pure-FTPd", Version: "1.0.50", Status: "stopped", Type: "FTP"},
	}
}

func (s *Service) Start(name string) error {
	_ = name
	return nil
}

func (s *Service) Stop(name string) error {
	_ = name
	return nil
}
