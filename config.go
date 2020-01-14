package main

import (
	"fmt"
	"net"
	"os"

	"github.com/castisdev/cilog"
	"github.com/spf13/viper"
)

// Server :
type Server struct {
	Sources             []string `mapstructure:"sources"`
	Destinations        []string `mapstructure:"destinations"`
	HeartbeatTimeoutSec uint     `mapstructure:"heartbeat_timeout_sec"`
	HeartbeatSleepSec   uint     `mapstructure:"heartbeat_sleep_sec"`
}

type Remover struct {
	RemoverSleepSec          uint `mapstructure:"remover_sleep_sec"`
	StorageUsageLimitPercent uint `mapstructure:"storage_usage_limit_percent"`
}

type Tasker struct {
	TaskerSleepSec   uint   `mapstructure:"tasker_sleep_sec"`
	TaskTimeout      int64  `mapstructure:"task_timeout_sec"`
	TaskCopySpeedBPS string `mapstructure:"task_copy_speed_bps"`
}

type Ignore struct {
	Prefixes []string `mapstructure:"prefixes"`
}

// Config :
type Config struct {
	SourceDirs          []string `mapstructure:"source_dirs"`
	HitcountHistoryFile string   `mapstructure:"hitcount_history_file"`
	GradeInfoFile       string   `mapstructure:"grade_info_file"`
	LogDir              string   `mapstructure:"log_dir"`
	LogLevel            string   `mapstructure:"log_level"`
	Servers             Server   `mapstructure:"servers"`
	WatchDir            string   `mapstructure:"watch_dir"`
	WatchIPString       string   `mapstructure:"watch_ip_string"`
	WatchTermMin        int      `mapstructure:"watch_term_min"`
	WatchHitBase        int      `mapstructure:"watch_hit_base"`
	EnableCoreDump      bool     `mapstructure:"enable_coredump"`
	ListenAddr          string   `mapstructure:"listen_addr"`
	Remover             Remover  `mapstructure:"remover"`
	Tasker              Tasker   `mapstructure:"tasker"`
	Ignore              Ignore   `mapstructure:"ignore"`
}

// ReadConfig :
func ReadConfig(configFile string) (*Config, error) {
	viper.SetDefault("log_dir", "log")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("enable_coredump", true)
	viper.SetDefault("tasker.task_timeout_sec", 3600)
	viper.SetDefault("tasker.task_copy_speed_bps", 10000000)
	viper.SetDefault("tasker.tasker_sleep_sec", 60)
	viper.SetDefault("listen_addr", "127.0.0.1:8080")
	viper.SetDefault("servers.heartbeat_timeout_sec", uint(5))
	viper.SetDefault("servers.heartbeat_sleep_sec", uint(30))
	viper.SetDefault("remover.remover_sleep_sec", uint(30))
	viper.SetDefault("remover.storage_usage_limit_percent", uint(90))

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
			fmt.Printf("not exist source dir (%s)\n", err)
			os.Exit(-1)
		}
	}

	if _, err := cilog.LevelFromString(c.LogLevel); err != nil {
		fmt.Printf("invalid log level : error(%s)", err)
	}

	if _, _, err := net.SplitHostPort(c.ListenAddr); err != nil {
		fmt.Printf("invalid listen_addr : error(%s)", err)
		os.Exit(-1)
	}
}
