package app

import (
	"log"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/core"
	"github.com/alireza0/s-ui/cronjob"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/service"
	"github.com/alireza0/s-ui/sub"
	"github.com/alireza0/s-ui/web"

	"github.com/op/go-logging"
)

type APP struct {
	service.SettingService
	configService *service.ConfigService
	webServer     *web.Server
	subServer     *sub.Server
	cronJob       *cronjob.CronJob
	logger        *logging.Logger
	core          *core.Core
}

func NewApp() *APP {
	return &APP{}
}

func (a *APP) Init() error {
	log.Printf("%v %v", config.GetName(), config.GetVersion())

	a.initLog()

	err := database.InitDB(config.GetDBPath())
	if err != nil {
		return err
	}

	// Init Setting
	a.SettingService.GetAllSetting()

	a.core = core.NewCore()

	a.cronJob = cronjob.NewCronJob()
	a.webServer = web.NewServer()
	a.subServer = sub.NewServer()

	a.configService = service.NewConfigService(a.core)

	return nil
}

func (a *APP) Start() error {
	loc, err := a.SettingService.GetTimeLocation()
	if err != nil {
		return err
	}

	trafficAge, err := a.SettingService.GetTrafficAge()
	if err != nil {
		return err
	}

	statsBucketSeconds, err := a.SettingService.GetStatsBucketSeconds()
	if err != nil {
		return err
	}

	globalReset, err := a.SettingService.GetGlobalReset()
	if err != nil {
		return err
	}

	err = a.cronJob.Start(loc, trafficAge, statsBucketSeconds, globalReset)
	if err != nil {
		return err
	}

	// 根据环境变量控制 web 服务启动
	if config.GetEnableWeb() {
		err = a.webServer.Start()
		if err != nil {
			return err
		}
		logger.Info("Web server started")
	} else {
		logger.Info("Web server disabled by SUI_ENABLE_WEB=false")
	}

	// 根据环境变量控制 sub 服务启动
	if config.GetEnableSub() {
		err = a.subServer.Start()
		if err != nil {
			return err
		}
		logger.Info("Sub server started")
	} else {
		logger.Info("Sub server disabled by SUI_ENABLE_SUB=false")
	}

	err = a.configService.StartCore()
	if err != nil {
		logger.Error(err)
	}

	return nil
}

func (a *APP) Stop() {
	a.cronJob.Stop()
	
	// 条件性停止 sub 服务
	if config.GetEnableSub() && a.subServer != nil {
		err := a.subServer.Stop()
		if err != nil {
			logger.Warning("stop Sub Server err:", err)
		}
	}
	
	// 条件性停止 web 服务
	if config.GetEnableWeb() && a.webServer != nil {
		err := a.webServer.Stop()
		if err != nil {
			logger.Warning("stop Web Server err:", err)
		}
	}
	
	err := a.configService.StopCore()
	if err != nil {
		logger.Warning("stop Core err:", err)
	}
}

func (a *APP) initLog() {
	switch config.GetLogLevel() {
	case config.Debug:
		logger.InitLogger(logging.DEBUG)
	case config.Info:
		logger.InitLogger(logging.INFO)
	case config.Warn:
		logger.InitLogger(logging.WARNING)
	case config.Error:
		logger.InitLogger(logging.ERROR)
	case config.Silent:
		logger.InitLogger(logging.WARNING)
	default:
		log.Fatal("unknown log level:", config.GetLogLevel())
	}
}

func (a *APP) RestartApp() {
	a.Stop()
	a.Start()
}

func (a *APP) GetCore() *core.Core {
	return a.core
}
