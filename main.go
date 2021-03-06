package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"time"

	"github.com/castisdev/cfm/api"
	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/fmfm"
	"github.com/castisdev/cfm/heartbeater"
	"github.com/castisdev/cfm/remover"
	"github.com/castisdev/cfm/tailer"
	"github.com/castisdev/cfm/tasker"
	"github.com/castisdev/cilog"
	"github.com/kardianos/osext"
)

// App constant
const (
	AppName      = "cfm"
	AppVersion   = "1.0.0"
	AppPreRelVer = "qr3"
)

func main() {
	debug.SetTraceback("crash")
	doCli()
	c := newConfig()
	enableCoreDump(c)
	configCiLogger(c)

	cilog.Infof("started main process")
	startHeartbeater(c)

	mgr := startManager(c)

	startHttpServer(c, mgr)
}

func doCli() {
	printSimpleVer := flag.Bool("v", false, "print version")
	printVer := flag.Bool("version", false, "print version includes pre-release version")
	flag.Parse()

	if *printSimpleVer {
		fmt.Println(AppName + " " + AppVersion)
		os.Exit(0)
	}

	if *printVer {
		fmt.Println(AppName + " " + AppVersion + "-" + AppPreRelVer)
		os.Exit(0)
	}
}

func newConfig() *Config {
	execDir, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatalf("failed to get executable folder, %s", err)
	}
	c, err := ReadConfig(path.Join(execDir, "cfm.yml"))
	if err != nil {
		log.Fatalf("failed to read config, error(%s)", err.Error())
	}
	err = ValidationConfig(*c)
	if err != nil {
		log.Fatalf("failed to read config, error(%s)", err.Error())
	}
	return c
}

func enableCoreDump(c *Config) {
	if c.EnableCoreDump {
		if err := common.EnableCoreDump(); err != nil {
			log.Fatalf("can not enable coredump, error(%s)", err.Error())
		}
	}
}

func configCiLogger(c *Config) {
	logLevel, _ := cilog.LevelFromString(c.LogLevel)

	mLogWriter := common.MLogWriter{
		LogWriter: cilog.NewLogWriter(c.LogDir, AppName, 10*1024*1024),
		Dir:       c.LogDir,
		App:       AppName,
		MaxSize:   (10 * 1024 * 1024)}

	cilog.Set(mLogWriter, AppName, AppVersion, logLevel)
}

func startHeartbeater(c *Config) {
	for _, s := range c.Servers.Sources {
		heartbeater.Add(s)
	}
	for _, s := range c.Servers.Destinations {
		heartbeater.Add(s)
	}
	heartbeater.SetTimoutSec(c.Servers.HeartbeatTimeoutSec)
	heartbeater.SetHeartbeatSec(c.Servers.HeartbeatSec)

	// tasker가 heartbeater의 결과를 사용하기 때문에,
	// heartbeater가 한 번 실행하고, tasker 가 실행될 수 있도록 한 번 실행시킴
	heartbeater.Heartbeat()
	go heartbeater.RunForever()
}

func startManager(c *Config) (manager *fmfm.Manager) {
	watcher := fmfm.NewWatcher(
		c.GradeInfoFile, c.HitcountHistoryFile,
		c.Watcher.FireInitialEvent, c.Watcher.EventTimeoutSec,
		c.Watcher.PollingSec)
	runner := fmfm.NewRunner(
		c.Runner.BetweenEventsRunSec, c.Runner.PeriodicRunSec,
		newRemover(c),
		newTasker(c),
		newTailer(c),
	)
	runner.SetupRuns = fmfm.ToSetupRuns(c.Runner.SetupRuns)

	manager = fmfm.NewManager(watcher, runner)

	go manager.Manage()
	return manager
}

func newTailer(c *Config) (tlr *tailer.Tailer) {
	tlr = tailer.NewTailer()
	tlr.SetWatchDir(c.WatchDir)
	tlr.SetWatchIPString(c.WatchIPString)
	tlr.SetWatchTermMin(c.WatchTermMin)
	tlr.SetWatchHitBase(c.WatchHitBase)
	return tlr
}

func newRemover(c *Config) (rmr *remover.Remover) {
	rmr = remover.NewRemover()
	for _, s := range c.Servers.Destinations {
		rmr.Servers.Add(s)
	}
	for _, s := range c.SourceDirs {
		rmr.SourcePath.Add(s)
	}
	rmr.SetSleepSec(c.Remover.RemoverSleepSec)
	if err := rmr.SetDiskUsageLimitPercent(
		c.Remover.StorageUsageLimitPercent); err != nil {
		log.Fatalf("can not configure remover. storage_usage_limit_percent"+
			", error(%s)", err.Error())
	}
	rmr.SetGradeInfoFile(c.GradeInfoFile)
	rmr.SetHitcountHistoryFile(c.HitcountHistoryFile)
	rmr.SetIgnorePrefixes(c.Ignore.Prefixes)
	rmr.Tail.SetWatchDir(c.WatchDir)
	rmr.Tail.SetWatchIPString(c.WatchIPString)
	rmr.Tail.SetWatchTermMin(c.WatchTermMin)
	rmr.Tail.SetWatchHitBase(c.WatchHitBase)

	return rmr
}

func newTasker(c *Config) (tskr *tasker.Tasker) {
	tskr = tasker.NewTasker()
	for _, s := range c.Servers.Sources {
		tskr.SrcServers.Add(s)
	}
	for _, s := range c.Servers.Destinations {
		tskr.DstServers.Add(s)
	}
	tskr.SetSleepSec(c.Tasker.TaskerSleepSec)
	tskr.SetTaskTimeout(time.Duration(c.Tasker.TaskTimeout) * time.Second)
	tskr.SetHitcountHistoryFile(c.HitcountHistoryFile)
	tskr.SetGradeInfoFile(c.GradeInfoFile)
	tskr.SetTaskCopySpeed(c.Tasker.TaskCopySpeedBPS)
	tskr.SetIgnorePrefixes(c.Ignore.Prefixes)
	tskr.Tail.SetWatchDir(c.WatchDir)
	tskr.Tail.SetWatchIPString(c.WatchIPString)
	tskr.Tail.SetWatchTermMin(c.WatchTermMin)
	tskr.Tail.SetWatchHitBase(c.WatchHitBase)

	for _, s := range c.SourceDirs {
		tskr.SourcePath.Add(s)
	}

	tskr.InitTasks()
	return tskr
}

func startHttpServer(c *Config, m *fmfm.Manager) {
	h := api.NewAPIHandler(m)
	router := api.NewRouter(h)
	s := &http.Server{
		Addr:         c.ListenAddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	err := s.ListenAndServe()
	if err != nil {
		log.Fatalf("failed to start, error(%s)", err.Error())
	}
}
