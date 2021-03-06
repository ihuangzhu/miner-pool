package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"miner-pool/config"
	"miner-pool/core"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

var daemon bool
var startCmd = &cobra.Command{
	Use:          "start",
	Short:        "Start the server",
	RunE:         startCmdF,
	SilenceUsage: true,
}

func init() {
	RootCmd.AddCommand(startCmd)
	startCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "run with daemon?")
	RootCmd.RunE = startCmdF
}

func startCmdF(cmd *cobra.Command, args []string) error {
	// 后台启动
	if daemon {
		runDaemon()
	}

	// 启动应用程序
	interruptChan := make(chan os.Signal, syscall.SIGHUP)
	// 加载配置文件
	cfg, err := loadConfig(cmd)
	if err != nil {
		log.Errorf("Error loading configuration: %v", err.Error())
	}
	// 启动服务器
	return runServer(cfg, interruptChan)
}

func runDaemon() {
	// 获取应用名
	app, dir := getAppDir()

	// 拿到启动命令并自启动
	bin := fmt.Sprintf("%s/%s", dir, app)
	command := exec.Command(bin, "start")
	command.Start()

	// 打印日志
	log.Infof("Server start, [PID] %d running...", command.Process.Pid)
	ioutil.WriteFile("mp.lock", []byte(fmt.Sprintf("%d", command.Process.Pid)), 0666)
	daemon = false
	os.Exit(0)
}

func runServer(cfg *config.Config, interruptChan chan os.Signal) error {
	initDebug(cfg.Debugger)
	initLogger(cfg.Logger)

	server := core.NewServer(cfg)
	defer server.Close()

	server.Start()

	// wait for kill signal before attempting to gracefully shutdown
	// the running service
	signal.Notify(interruptChan, syscall.SIGINT, syscall.SIGTERM)
	<-interruptChan

	return nil
}

func initDebug(cfg *config.Debugger) {
	if *cfg.Enable {
		go http.ListenAndServe(*cfg.Listen, nil)
	}
}

func initLogger(cfg *config.Logger) {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.TextFormatter{})

	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)

	if *cfg.Mode == "file" {
		// You could set this to any `io.Writer` such as a file
		file, err := os.OpenFile(*cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
		} else {
			log.Info("Failed to log to file, using default stderr")
		}
	}

	// Only log the warning severity or above.
	log.SetLevel(log.Level(*cfg.Level))
}
