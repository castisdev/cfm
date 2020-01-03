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

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/heartbeater"
	"github.com/castisdev/cfm/remover"
	"github.com/castisdev/cfm/tasker"
	"github.com/castisdev/cilog"
	"github.com/kardianos/osext"
)

var tasks *tasker.Tasks
var api common.MLogger

// App constant
const (
	AppName      = "cfm"
	AppVersion   = "1.0.0"
	AppPreRelVer = "qr2"
)

func main() {

	debug.SetTraceback("crash")

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

	execDir, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatalf("fail to get executable folder, %s", err)
	}

	c, err := ReadConfig(path.Join(execDir, "cfm.yml"))
	if err != nil {
		log.Fatalf("fail to read config, error(%s)", err)
	}

	ValidationConfig(*c)

	if c.EnableCoreDump {
		if err := common.EnableCoreDump(); err != nil {
			log.Fatalf("can not enable coredump, error(%s)", err.Error())
		}
	}

	logLevel, _ := cilog.LevelFromString(c.LogLevel)

	mLogWriter := common.MLogWriter{
		LogWriter: cilog.NewLogWriter(c.LogDir, AppName, 10*1024*1024),
		Dir:       c.LogDir,
		App:       AppName,
		MaxSize:   (10 * 1024 * 1024)}

	api = common.MLogger{
		Logger: cilog.StdLogger(),
		Mod:    "api"}

	cilog.Set(mLogWriter, AppName, AppVersion, logLevel)
	cilog.Infof("process start")

	for _, s := range c.Servers.Sources {
		heartbeater.Add(s)
	}
	for _, s := range c.Servers.Destinations {
		heartbeater.Add(s)
	}
	heartbeater.SetTimoutSec(c.Servers.HeartbeatTimeoutSec)
	heartbeater.SetSleepSec(c.Servers.HeartbeatSleepSec)

	// tasker가 heartbeater의 결과를 사용하기 때문에,
	// heartbeater가 한 번 실행하고, tasker 가 실행될 수 있도록 한 번 실행시킴
	heartbeater.Heartbeat()
	go heartbeater.RunForever()

	for _, s := range c.Servers.Destinations {
		remover.Servers.Add(s)
	}
	for _, s := range c.SourceDirs {
		remover.SourcePath.Add(s)
	}
	remover.SetSleepSec(c.Remover.RemoverSleepSec)
	if err := remover.SetDiskUsageLimitPercent(
		c.Remover.StorageUsageLimitPercent); err != nil {
		log.Fatalf("can not configure remover.storage_usage_limit_percent"+
			", error(%s)", err.Error())
	}
	remover.SetGradeInfoFile(c.GradeInfoFile)
	remover.SetHitcountHistoryFile(c.HitcountHistoryFile)
	remover.SetAdvPrefix(c.AdvPrefixes)
	remover.Tail.SetWatchDir(c.WatchDir)
	remover.Tail.SetWatchIPString(c.WatchIPString)
	remover.Tail.SetWatchTermMin(c.WatchTermMin)
	remover.Tail.SetWatchHitBase(c.WatchHitBase)

	go remover.RunForever()

	for _, s := range c.Servers.Sources {
		tasker.SrcServers.Add(s)
	}

	for _, s := range c.Servers.Destinations {
		tasker.DstServers.Add(s)
	}

	tasker.SetSleepSec(c.Tasker.TaskerSleepSec)
	tasker.SetTaskTimeout(time.Duration(c.Tasker.TaskTimeout) * time.Second)
	tasker.SetHitcountHistoryFile(c.HitcountHistoryFile)
	tasker.SetGradeInfoFile(c.GradeInfoFile)
	tasker.SetTaskCopySpeed(c.Tasker.TaskCopySpeedBPS)
	tasker.SetAdvPrefix(c.AdvPrefixes)
	tasker.Tail.SetWatchDir(c.WatchDir)
	tasker.Tail.SetWatchIPString(c.WatchIPString)
	tasker.Tail.SetWatchTermMin(c.WatchTermMin)
	tasker.Tail.SetWatchHitBase(c.WatchHitBase)

	for _, s := range c.SourceDirs {
		tasker.SourcePath.Add(s)
	}

	go tasker.RunForever()

	tasks = tasker.GetTaskListInstance()

	router := NewRouter()
	s := &http.Server{
		Addr:         c.ListenAddr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	s.ListenAndServe()

}
