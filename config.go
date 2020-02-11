package main

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/castisdev/cfm/fmfm"
	"github.com/castisdev/cilog"
	"github.com/spf13/viper"
)

// Server :
type Server struct {
	Sources             []string `mapstructure:"sources"`
	Destinations        []string `mapstructure:"destinations"`
	HeartbeatTimeoutSec uint     `mapstructure:"heartbeat_timeout_sec"`
	HeartbeatSec        uint     `mapstructure:"heartbeat_interval_sec"`
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

type Watcher struct {
	FireInitialEvent bool   `mapstructure:"fire_initial_event"`
	EventTimeoutSec  uint32 `mapstructure:"event_timeout_sec"`
	PollingSec       uint32 `mapstructure:"poll_interval_sec"`
}

type Runner struct {
	BetweenEventsRunSec uint32              `mapstructure:"between_events_run_interval_sec"`
	PeriodicRunSec      uint32              `mapstructure:"periodic_run_interval_sec"`
	SetupRuns           map[string][]string `mapstructure:"setup_runs"`
}

func (r *Runner) validate() error {
	for rs, runs := range r.SetupRuns {
		if fmfm.ToRuns(rs) == 0 {
			return errors.New(
				fmt.Sprintf("%s in runner.setup_runs:, invalid name of runs", rs))
		}
		for _, run := range runs {
			if fmfm.ToRun(run) == 0 {
				return errors.New(
					fmt.Sprintf("%s in runner.setup_runs.%s:, invalid run description", run, rs))
			}
		}
	}
	return nil
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
	Watcher             Watcher  `mapstructure:"watcher"`
	Runner              Runner   `mapstructure:"runner"`
}

// ReadConfig :
func ReadConfig(configFile string) (*Config, error) {
	viper.SetDefault("log_dir", "log")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("servers.heartbeat_timeout_sec", uint(5))
	viper.SetDefault("servers.heartbeat_interval_sec", uint(30))
	viper.SetDefault("enable_coredump", true)
	viper.SetDefault("listen_addr", "127.0.0.1:8080")
	viper.SetDefault("remover.remover_sleep_sec", uint(30))
	viper.SetDefault("remover.storage_usage_limit_percent", uint(90))
	viper.SetDefault("tasker.tasker_sleep_sec", 60)
	viper.SetDefault("tasker.task_timeout_sec", 3600)
	viper.SetDefault("tasker.task_copy_speed_bps", "10000000")
	viper.SetDefault("watcher.fire_initial_event", true)
	viper.SetDefault("watcher.event_timeout_sec", uint32(600))
	viper.SetDefault("watcher.poll_interval_sec", uint32(60))
	viper.SetDefault("runner.between_events_run_interval_sec", uint32(10))
	viper.SetDefault("runner.periodic_run_interval_sec", uint32(0))

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
func ValidationConfig(c Config) error {

	// Source 경로가 존재하지 않을 경우 프로세스 중지
	for _, sdir := range c.SourceDirs {
		if _, err := os.Stat(sdir); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("invalid source_dirs : error(%s)", err))
		}
	}

	if _, err := cilog.LevelFromString(c.LogLevel); err != nil {
		return errors.New(fmt.Sprintf("invalid log_level : error(%s)", err))
	}

	if _, _, err := net.SplitHostPort(c.ListenAddr); err != nil {
		return errors.New(fmt.Sprintf("invalid listen_addr : error(%s)", err))
	}

	if err := c.Runner.validate(); err != nil {
		return errors.New(fmt.Sprintf("invalid runner : error(%s)", err))
	}

	return nil
}
