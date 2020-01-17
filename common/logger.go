package common

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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

	var where string
	var file string
	var line int
	var ok bool
	var pc uintptr
	var pkg string
	var fn string
	pc, file, line, ok = runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
		pkg = "???"
		fn = "???"
	} else {
		file = filepath.Base(file)
		pkg = cilog.PackageBase(runtime.FuncForPC(pc).Name())
		fn = filepath.Base(runtime.FuncForPC(pc).Name())
		fn = fn[strings.Index(fn, ".")+1:]
	}
	where = pkg + "::" + file + "::" + fn + ":" + strconv.Itoa(line)

	if lvl == cilog.DEBUG {
		var pfile string
		var pline int
		var pok bool
		var ppc uintptr
		var ppkg string
		var pfn string
		ppc, pfile, pline, pok = runtime.Caller(calldepth + 1)
		if !pok {
			pfn = ""
		} else {
			pfile = filepath.Base(pfile)
			ppkg = cilog.PackageBase(runtime.FuncForPC(ppc).Name())
			pfn = filepath.Base(runtime.FuncForPC(ppc).Name())
			pfn = pfn[strings.Index(pfn, ".")+1:]
			if ppkg != pkg || pfile != file {
				where = ppkg + "::" + pfile + "::" + pfn + ":" + strconv.Itoa(pline) +
					"|" + where
			} else {
				where = pkg + "::" + file + "::" +
					pfn + ":" + strconv.Itoa(pline) + "|" + fn + ":" + strconv.Itoa(line)
			}
		}
	}

	m := l.GetModule() + "," + l.GetModuleVer() + "," + timeStr + "," +
		lvl.Output() + "," + where + "," + l.Mod + "," + msg
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
