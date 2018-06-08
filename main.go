package main

import (
	"net/http"
	"time"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cfm/remover"
	"github.com/castisdev/cfm/tasker"
	"github.com/castisdev/cilog"
)

var tasks *tasker.Tasks

func main() {

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

	cilog.Set(cilog.NewLogWriter(c.LogDir, moduleName, 10*1024*1024), moduleName, moduleVersion, logLevel)

	for _, s := range c.Servers.Destinations {
		remover.Servers.Add(s)
	}
	remover.SetDiskUsageLimitPercent(c.StorageUsageLimitPercent)

	for _, s := range c.SourceDirs {
		remover.SourcePath.Add(s)
	}

	//go remover.RunForever()

	for _, s := range c.Servers.Sources {
		tasker.SrcServers.Add(s)
	}

	for _, s := range c.Servers.Destinations {
		tasker.DstServers.Add(s)
	}

	tasker.SetTaskTimeout(time.Duration(c.TaskTimeout) * time.Second)
	tasker.SetHitcountHistoryFile(c.HitcountHistoryFile)
	tasker.SetGradeInfoFile(c.GradeInfoFile)

	for _, s := range c.SourceDirs {
		tasker.SourcePath.Add(s)
	}

	go tasker.RunForever()

	tasks = tasker.GetTaskListInstance()

	// dummy data
	//tasks.CreateTask(&tasker.Task{FilePath: "/data2/a.mpg", FileName: "a.mpg", SrcIP: "127.0.0.1"})
	//tasks.CreateTask(&tasker.Task{FilePath: "/data3/b.mpg", FileName: "b.mpg", SrcIP: "127.0.0.2"})
	//tasks.CreateTask(&tasker.Task{FilePath: "/data2/c.mpg", FileName: "c.mpg", SrcIP: "127.0.0.3"})

	router := NewRouter()
	s := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	s.ListenAndServe()

}
