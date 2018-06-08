package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/remover"
	"github.com/castisdev/cfm/tasker"
	"github.com/castisdev/cilog"
)

var tasks *tasker.Tasks

// App constant
const (
	AppName      = "cfm"
	AppVersion   = "1.0.0"
	AppPreRelVer = "-rc1"
)

func main() {

	printSimpleVer := flag.Bool("v", false, "print version")
	printVer := flag.Bool("version", false, "print version includes pre-release version")
	flag.Parse()

	if *printSimpleVer {
		fmt.Println(AppName + " " + AppVersion)
		os.Exit(0)
	}

	if *printVer {
		fmt.Println(AppName + " " + AppVersion + AppPreRelVer)
		os.Exit(0)
	}

	c, err := ReadConfig("cfm.yml")
	if err != nil {
		panic(err)
	}

	ValidationConfig(*c)

	if c.EnableCoreDump {
		if err := common.EnableCoreDump(); err != nil {
			log.Fatalf("can not enable coredump,error(%s)", err.Error())
		}
	}

	logLevel, _ := cilog.LevelFromString(c.LogLevel)

	cilog.Set(cilog.NewLogWriter(c.LogDir, AppName, 10*1024*1024), AppName, AppVersion, logLevel)
	cilog.Infof("process start")

	for _, s := range c.Servers.Destinations {
		remover.Servers.Add(s)
	}

	for _, s := range c.SourceDirs {
		remover.SourcePath.Add(s)
	}

	remover.SetDiskUsageLimitPercent(c.StorageUsageLimitPercent)
	remover.SetGradeInfoFile(c.GradeInfoFile)
	remover.SetHitcountHistoryFile(c.HitcountHistoryFile)
	remover.SetAdvPrefix(c.AdvPrefixes)

	go remover.RunForever()

	for _, s := range c.Servers.Sources {
		tasker.SrcServers.Add(s)
	}

	for _, s := range c.Servers.Destinations {
		tasker.DstServers.Add(s)
	}

	tasker.SetTaskTimeout(time.Duration(c.TaskTimeout) * time.Second)
	tasker.SetHitcountHistoryFile(c.HitcountHistoryFile)
	tasker.SetGradeInfoFile(c.GradeInfoFile)
	tasker.SetTaskCopySpeed(c.TaskCopySpeedBPS)
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
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	s.ListenAndServe()

}
