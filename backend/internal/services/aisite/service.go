package aisite

import (
	"github.com/open-panel/open-panel/internal/services/aichat"
	"github.com/open-panel/open-panel/internal/services/appstore"
	"github.com/open-panel/open-panel/internal/services/cron"
	"github.com/open-panel/open-panel/internal/services/devops"
	"github.com/open-panel/open-panel/internal/services/nodejs"
	"github.com/open-panel/open-panel/internal/services/settings"
	"github.com/open-panel/open-panel/internal/services/website"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	dataDir  string
	website  *website.Service
	devops   *devops.Service
	aichat   *aichat.Service
	appstore *appstore.Service
	settings *settings.Service
	nodejs   *nodejs.Service
	cron     *cron.Service
}

func NewService(
	db *gorm.DB,
	dataDir string,
	websiteSvc *website.Service,
	devopsSvc *devops.Service,
	aichatSvc *aichat.Service,
	appSvc *appstore.Service,
	settingsSvc *settings.Service,
	nodejsSvc *nodejs.Service,
	cronSvc *cron.Service,
) *Service {
	return &Service{
		db:       db,
		dataDir:  dataDir,
		website:  websiteSvc,
		devops:   devopsSvc,
		aichat:   aichatSvc,
		appstore: appSvc,
		settings: settingsSvc,
		nodejs:   nodejsSvc,
		cron:     cronSvc,
	}
}
