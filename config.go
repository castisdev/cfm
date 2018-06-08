package main

import (
	"fmt"
	"os"

	"github.com/castisdev/cilog"
	"github.com/spf13/viper"
)

// Server :
type Server struct {
	Sources      []string `mapstructure:"sources"`
	Destinations []string `mapstructure:"destinations"`
}

// Config :
type Config struct {
	AdvPrefixes              []string `mapstructure:"adv_prefixes"`
	SourceDirs               []string `mapstructure:"source_dirs"`
	HitcountHistoryFile      string   `mapstructure:"hitcount_history_file"`
	GradeInfoFile            string   `mapstructure:"grade_info_file"`
	StorageUsageLimitPercent int      `mapstructure:"storage_usage_limit_percent"`
	LogDir                   string   `mapstructure:"log_dir"`
	LogLevel                 string   `mapstructure:"log_level"`
	Servers                  Server   `mapstructure:"servers"`
	TaskTimeout              int64    `mapstructure:"task_timeout_sec"`
	EnableCoreDump           bool     `mapstructure:"enable_coredump"`
}

// ReadConfig :
func ReadConfig(configFile string) (*Config, error) {

	viper.SetDefault("enable_coredump", false)
	var c Config
	viper.SetConfigFile(configFile)

	if err := viper.ReadInConfig(); err != nil {
		return &Config{}, err
	}

	if err := viper.Unmarshal(&c); err != nil {
		return &Config{}, err
	}

	return &c, nil
}

// ValidationConfig :
func ValidationConfig(c Config) {

	// Source 경로가 존재하지 않을 경우 프로세스 중지
	for _, sdir := range c.SourceDirs {
		if _, err := os.Stat(sdir); os.IsNotExist(err) {
			fmt.Printf("not exists source dir (%s)\n", err)
			os.Exit(-1)
		}
	}

	if _, err := cilog.LevelFromString(c.LogLevel); err != nil {
		fmt.Printf("invalid log level : error(%s)", err)
	}

}
