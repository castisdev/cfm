package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func writeconfigfile(dir string, filename string, yml []byte) {
	fp := filepath.Join(dir, filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.Create(fp)
	if err != nil {
		f.Close()
		log.Fatal(err)
	}
	fmt.Fprintf(f, "%s", yml)
	f.Close()
}

func deletefile(dir string, filename string) {
	dir = filepath.Clean(dir)
	if dir == "." || dir == ".." {
		log.Fatal(errors.New("do not delete current or parent folder"))
	}
	fp := filepath.Join(dir, filename)
	err := os.RemoveAll(fp)
	if err != nil {
		log.Fatal(err)
	}
}

func TestConfigWatcher(t *testing.T) {
	viper.SetConfigType("yaml")
	var tctbl = []struct {
		yml  []byte
		wfe  bool
		wets uint32
		wpis uint32
	}{
		{yml: []byte(`
  watcher:
    fire_initial_event: true
    event_timeout_sec : 30
    poll_interval_sec : 60
  `), wfe: true, wets: 30, wpis: 60},
		{yml: []byte(`
  watcher:
    fire_initial_event: false
    event_timeout_sec : 0
    poll_interval_sec : 0
  `), wfe: false, wets: 0, wpis: 0},
	}

	for _, tc := range tctbl {
		var c Config
		if err := viper.ReadConfig(bytes.NewBuffer(tc.yml)); err != nil {
			t.Error(err)
		}
		if err := viper.Unmarshal(&c); err != nil {
			t.Error(err)
		}
		assert.Equal(t, tc.wfe, c.Watcher.FireInitialEvent)
		assert.Equal(t, tc.wets, c.Watcher.EventTimeoutSec)
		assert.Equal(t, tc.wpis, c.Watcher.PollingSec)
	}

	dir := "testconfig"
	file := "config.yml"
	defer deletefile(dir, "")
	for _, tc := range tctbl {
		var c Config
		writeconfigfile(dir, file, tc.yml)
		defer deletefile(dir, file)
		viper.SetConfigFile(filepath.Join(dir, file))
		if err := viper.ReadInConfig(); err != nil {
			t.Error(err)
		}
		if err := viper.Unmarshal(&c); err != nil {
			t.Error(err)
		}
		assert.Equal(t, tc.wfe, c.Watcher.FireInitialEvent)
		assert.Equal(t, tc.wets, c.Watcher.EventTimeoutSec)
		assert.Equal(t, tc.wpis, c.Watcher.PollingSec)
	}
}

func TestConfigRunner(t *testing.T) {
	viper.SetConfigType("yaml")
	var tctbl = []struct {
		yml    []byte
		wberis uint32
		wpris  uint32
		wsetup map[string][]string
		wvalid bool
	}{
		{yml: []byte(`
  runner:
    between_events_run_interval_sec: 10
    periodic_run_interval_sec : 40
  `), wberis: 10, wpris: 40, wvalid: true},
		{yml: []byte(`
   runner:
    between_events_run_interval_sec: 0
    periodic_run_interval_sec : 0
  `), wberis: 0, wpris: 0, wvalid: true},
		{yml: []byte(`
  runner:
    between_events_run_interval_sec: 0
    periodic_run_interval_sec : 0
    setup_runs:
      eventRuns: [nop]
  `), wberis: 0, wpris: 0,
			wsetup: map[string][]string{"eventruns": []string{"nop"}}, wvalid: true,
		},
		{yml: []byte(`
      runner:
        between_events_run_interval_sec: 0
        periodic_run_interval_sec : 0
        setup_runs:
          eventRuns: [makeFmm, makeRisingHit, runRemover, runTasker]
          eventTimeoutRuns: [makeFmm, makeRisingHit, runRemover, runTasker]
          betweenEventsRuns: [makeRisingHit, runRemover, runTasker]
          periodicRuns: [nop]
    `), wberis: 0, wpris: 0,
			wsetup: map[string][]string{
				"eventruns":         []string{"makeFmm", "makeRisingHit", "runRemover", "runTasker"},
				"eventtimeoutruns":  []string{"makeFmm", "makeRisingHit", "runRemover", "runTasker"},
				"betweeneventsruns": []string{"makeRisingHit", "runRemover", "runTasker"},
				"periodicruns":      []string{"nop"},
			},
			wvalid: true,
		},
		{yml: []byte(`
      runner:
        between_events_run_interval_sec: 0
        periodic_run_interval_sec : 0
        setup_runs:
          eventRuns: [nop]
          eventTimeoutRuns: [makeFmm, makeRisingHit, runRemover, runTasker]
          someRuns: [nop]
    `), wberis: 0, wpris: 0,
			wsetup: map[string][]string{
				"eventruns":        []string{"nop"},
				"eventtimeoutruns": []string{"makeFmm", "makeRisingHit", "runRemover", "runTasker"},
				"someruns":         []string{"nop"}},
			wvalid: false,
		},
		{yml: []byte(`
      runner:
        between_events_run_interval_sec: 0
        periodic_run_interval_sec : 0
        setup_runs:
          eventRuns: [someRun]
          eventTimeoutRuns: [makeFmm, makeRisingHit, runRemover, runTasker]
    `), wberis: 0, wpris: 0,
			wsetup: map[string][]string{
				"eventruns":        []string{"someRun"},
				"eventtimeoutruns": []string{"makeFmm", "makeRisingHit", "runRemover", "runTasker"},
			},
			wvalid: false,
		},
	}

	for _, tc := range tctbl {
		var c Config
		if err := viper.ReadConfig(bytes.NewBuffer(tc.yml)); err != nil {
			t.Error(err)
		}
		if err := viper.Unmarshal(&c); err != nil {
			t.Error(err)
		}
		assert.Equal(t, tc.wberis, c.Runner.BetweenEventsRunSec)
		assert.Equal(t, tc.wpris, c.Runner.PeriodicRunSec)
		assert.Equal(t, tc.wsetup, c.Runner.SetupRuns)
		if err := c.Runner.validate(); err != nil {
			assert.Equal(t, tc.wvalid, false)
		} else {
			assert.Equal(t, tc.wvalid, true)
		}
	}

	dir := "testconfig"
	file := "config.yml"
	defer deletefile(dir, "")
	for _, tc := range tctbl {
		var c Config
		writeconfigfile(dir, file, tc.yml)
		defer deletefile(dir, file)
		viper.SetConfigFile(filepath.Join(dir, file))
		if err := viper.ReadInConfig(); err != nil {
			t.Error(err)
		}
		if err := viper.Unmarshal(&c); err != nil {
			t.Error(err)
		}
		assert.Equal(t, tc.wberis, c.Runner.BetweenEventsRunSec)
		assert.Equal(t, tc.wpris, c.Runner.PeriodicRunSec)
		assert.Equal(t, tc.wsetup, c.Runner.SetupRuns)
		if err := c.Runner.validate(); err != nil {
			assert.Equal(t, tc.wvalid, false)
		} else {
			assert.Equal(t, tc.wvalid, true)
		}
	}
}
