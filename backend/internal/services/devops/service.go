package devops

import (
	"github.com/open-panel/open-panel/internal/services/appstore"
	"github.com/open-panel/open-panel/internal/services/compose"
	"github.com/open-panel/open-panel/internal/services/settings"
	"github.com/open-panel/open-panel/internal/services/webserver"
	"gorm.io/gorm"
)

type Service struct {
	db        *gorm.DB
	dataDir   string
	compose   *compose.Service
	webserver *webserver.Manager
	appstore  *appstore.Service
	settings  *settings.Service
}

func NewService(
	db *gorm.DB,
	dataDir string,
	composeSvc *compose.Service,
	ws *webserver.Manager,
	appSvc *appstore.Service,
	settingsSvc *settings.Service,
) *Service {
	return &Service{
		db:        db,
		dataDir:   dataDir,
		compose:   composeSvc,
		webserver: ws,
		appstore:  appSvc,
		settings:  settingsSvc,
	}
}
