package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os/exec"
)

var stopCmd = &cobra.Command{
	Use:          "stop",
	Short:        "Stop the server",
	RunE:         stopCmdF,
	SilenceUsage: true,
}

func init() {
	RootCmd.AddCommand(stopCmd)
}

func stopCmdF(cmd *cobra.Command, args []string) error {
	// 获取应用名
	_, dir := getAppDir()

	// 关闭服务器
	file := fmt.Sprintf("%s/%s", dir, "mp.lock")
	pid, _ := ioutil.ReadFile(file)
	command := exec.Command("kill", string(pid))
	command.Start()
	log.Infof("Server stop, [PID] %s running...", string(pid))

	return nil
}
