package cmd

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"miner-pool/config"
	"os"
	"path/filepath"
	"strings"
)

func getAppDir() (string, string) {
	app := strings.TrimLeft(os.Args[0], "./")
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Panic(err)
	}
	return app, dir
}

func getConfigPath(command *cobra.Command) string {
	configPath, _ := command.Flags().GetString("config")

	if configPath == "" {
		configPath = "config.json"
	}

	return configPath
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	configsPath := getConfigPath(cmd)
	if configsPath == "" {
		return nil, nil
	}

	file, err := os.Open(configsPath)
	if err != nil {
		return nil, fmt.Errorf("Unable to open configs file at %q: %w", configsPath, err)
	}
	defer file.Close()

	var customDefaults *config.Config
	err = json.NewDecoder(file).Decode(&customDefaults)
	if err != nil {
		return nil, fmt.Errorf("Unable to decode configs configuration: %w", err)
	}

	return customDefaults, nil
}
