package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	"github.com/alireza0/s-ui/agent"
	"github.com/alireza0/s-ui/app"
	"github.com/alireza0/s-ui/cmd"
	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/logger"

	"github.com/op/go-logging"
)

func runApp() {
	app := app.NewApp()

	err := app.Init()
	if err != nil {
		log.Fatal(err)
	}

	err = app.Start()
	if err != nil {
		log.Fatal(err)
	}

	sigCh := make(chan os.Signal, 1)
	// Trap shutdown signals
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM)
	for {
		sig := <-sigCh

		switch sig {
		case syscall.SIGHUP:
			app.RestartApp()
		default:
			app.Stop()
			return
		}
	}
}

func runAgent() {
	// Initialize logger for agent mode
	initAgentLogger()

	ag := agent.NewAgent()

	err := ag.Init()
	if err != nil {
		log.Fatal(err)
	}

	err = ag.Start()
	if err != nil {
		log.Fatal(err)
	}

	sigCh := make(chan os.Signal, 1)
	// Trap shutdown signals
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM)
	for {
		sig := <-sigCh

		switch sig {
		case syscall.SIGHUP:
			// Agent doesn't support restart via signal
			log.Println("SIGHUP received, agent does not support restart")
		default:
			ag.Stop()
			return
		}
	}
}

func main() {
	// Check for command line arguments first
	if len(os.Args) >= 2 {
		cmd.ParseCmd()
		return
	}

	// Check run mode
	if config.IsAgent() {
		log.Printf("%v %v (Agent Mode)", config.GetName(), config.GetVersion())
		runAgent()
	} else {
		log.Printf("%v %v (Panel Mode)", config.GetName(), config.GetVersion())
		runApp()
	}
}

func initAgentLogger() {
	switch config.GetLogLevel() {
	case config.Debug:
		logger.InitLogger(logging.DEBUG)
	case config.Info:
		logger.InitLogger(logging.INFO)
	case config.Warn:
		logger.InitLogger(logging.WARNING)
	case config.Error:
		logger.InitLogger(logging.ERROR)
	default:
		logger.InitLogger(logging.INFO)
	}
}
