package common

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/castisdev/cilog"
)

type MLogWriter struct {
	*cilog.LogWriter
	Dir     string
	App     string
	MaxSize int64
}

// Write :
func (w MLogWriter) Write(output []byte) (int, error) {
	t := time.Now()
	p := cilog.LogPath(w.Dir, w.App, w.MaxSize, t)
	n, err := w.LogWriter.WriteWithTime(output, t)
	symlinkbase := w.Dir
	p, _ = filepath.Rel(w.Dir, p)
	symlink := filepath.Join(symlinkbase, w.App+".log")
	if _, err := os.Lstat(symlink); err == nil {
		os.Remove(symlink)
	}
	os.Symlink(p, symlink)
	return n, err
}

type MLogger struct {
	*cilog.Logger
	Mod string
}

func (l *MLogger) Log(calldepth int, lvl cilog.Level, msg string, t time.Time) {
	if l.GetMinLevel() > lvl {
		return
	}
	timeStr := t.Format("2006-01-02,15:04:05.000000")
	var file string
	var line int
	var ok bool
	var pc uintptr
	var pkg string
	pc, file, line, ok = runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
		pkg = "???"
	} else {
		file = filepath.Base(file)
		pkg = cilog.PackageBase(runtime.FuncForPC(pc).Name())
	}

	m := l.GetModule() + "," + l.GetModuleVer() + "," + timeStr + "," +
		lvl.Output() + "," + pkg + "::" + file + ":" + strconv.Itoa(line) + "," +
		l.Mod + "," + msg
	if len(msg) == 0 || msg[len(msg)-1] != '\n' {
		m += "\n"
	}
	l.GetWriter().Write([]byte(m))
}

// Debugf :
func (l *MLogger) Debugf(format string, v ...interface{}) {
	l.Log(2, cilog.DEBUG, fmt.Sprintf(format, v...), time.Now())
}

// Reportf :
func (l *MLogger) Reportf(format string, v ...interface{}) {
	l.Log(2, cilog.REPORT, fmt.Sprintf(format, v...), time.Now())
}

// Infof :
func (l *MLogger) Infof(format string, v ...interface{}) {
	l.Log(2, cilog.INFO, fmt.Sprintf(format, v...), time.Now())
}

// Successf :
func (l *MLogger) Successf(format string, v ...interface{}) {
	l.Log(2, cilog.SUCCESS, fmt.Sprintf(format, v...), time.Now())
}

// Warningf :
func (l *MLogger) Warningf(format string, v ...interface{}) {
	l.Log(2, cilog.WARNING, fmt.Sprintf(format, v...), time.Now())
}

// Errorf :
func (l *MLogger) Errorf(format string, v ...interface{}) {
	l.Log(2, cilog.ERROR, fmt.Sprintf(format, v...), time.Now())
}

// Failf :
func (l *MLogger) Failf(format string, v ...interface{}) {
	l.Log(2, cilog.FAIL, fmt.Sprintf(format, v...), time.Now())
}

// Exceptionf :
func (l *MLogger) Exceptionf(format string, v ...interface{}) {
	l.Log(2, cilog.EXCEPTION, fmt.Sprintf(format, v...), time.Now())
}

// Criticalf :
func (l *MLogger) Criticalf(format string, v ...interface{}) {
	l.Log(2, cilog.CRITICAL, fmt.Sprintf(format, v...), time.Now())
}
