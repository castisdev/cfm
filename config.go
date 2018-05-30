package main

import (
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
}

// ReadConfig :
func ReadConfig(configFile string) (*Config, error) {

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
