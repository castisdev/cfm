package fmfm

import (
	"os"
	"testing"
	"time"

	"github.com/castisdev/cfm/myinotify"
	"github.com/stretchr/testify/assert"
)

func TestNewFileMonitor(t *testing.T) {
	now := time.Now()
	tctbl := []struct {
		path  string
		expfm FileMonitor
	}{
		{path: "./d1/d2/file",
			expfm: FileMonitor{
				Event:    myinotify.Event{Name: "./d1/d2/file", Op: 0},
				FilePath: "./d1/d2/file",
				Dir:      "d1/d2",
				Name:     "file",
				Dirs:     []string{"d1/d2", "d1", "."},
				Exist:    false,
				Mtime:    now,
			}},
		{path: "d1/d2/file",
			expfm: FileMonitor{
				Event:    myinotify.Event{Name: "d1/d2/file", Op: 0},
				FilePath: "d1/d2/file",
				Dir:      "d1/d2",
				Name:     "file",
				Dirs:     []string{"d1/d2", "d1", "."},
				Exist:    false,
				Mtime:    now,
			}},
		{path: "/d1/d2/file",
			expfm: FileMonitor{
				Event:    myinotify.Event{Name: "/d1/d2/file", Op: 0},
				FilePath: "/d1/d2/file",
				Dir:      "/d1/d2",
				Name:     "file",
				Dirs:     []string{"/d1/d2", "/d1", "/"},
				Exist:    false,
				Mtime:    now,
			}},
	}
	for _, tc := range tctbl {
		fm := NewFileMonitor(tc.path)
		fm.Mtime = now
		assert.Equal(t, tc.expfm, *fm)
	}
}

func TestNewDirs(t *testing.T) {
	tctbl := []struct {
		path    string
		expdirs []string
	}{
		{path: "d1/d2/file", expdirs: []string{"d1/d2", "d1", "."}},
		{path: "./d1/d2/file", expdirs: []string{"d1/d2", "d1", "."}},
		{path: "../d1/d2/file", expdirs: []string{"../d1/d2", "../d1", ".."}},
		{path: "/d1/d2/file", expdirs: []string{"/d1/d2", "/d1", "/"}},
		// 잘못된 path지만, error 처리가 안됨
		{path: ".../d1/d2/file", expdirs: []string{".../d1/d2", ".../d1", "...", "."}},
		// file 이 없어도 directory parsing이 됨
		{path: "d1/d2/", expdirs: []string{"d1/d2", "d1", "."}},
		{path: "/d1/d2/", expdirs: []string{"/d1/d2", "/d1", "/"}},
	}
	for _, tc := range tctbl {
		dirs := newDirs(tc.path)
		assert.Equal(t, tc.expdirs, dirs)
	}
}

func TestFileMonitorUpdateAndReset(t *testing.T) {
	tctbl := []struct {
		event   myinotify.Op
		exist   bool
		updated bool
	}{
		{event: myinotify.Create, exist: true, updated: false},
		{event: myinotify.Write, exist: true, updated: true},
		{event: myinotify.Remove, exist: false, updated: false},
		{event: myinotify.Chmod, exist: true, updated: true},
		{event: 0, updated: false},
	}

	f := NewFileMonitor("d1/file")
	for _, tc := range tctbl {
		f.Update(tc.event)
		if tc.event != 0 {
			assert.Equal(t, tc.exist, f.Exist)
			assert.Equal(t, tc.updated, f.Updated)
		} else {
			assert.Equal(t, tc.updated, f.Updated)
		}

		f.ResetUpdate()
		assert.Equal(t, false, f.Updated)
		assert.Equal(t, myinotifyUnknown, f.Event.Op)
	}
}

func TestFileMonitorUpdateMtimeAndResetUpadte(t *testing.T) {
	createfile("testwatcher", "exist")
	defer deletefile("testwatcher", "")
	fi, _ := os.Stat("testwatcher/exist")
	fimtime := fi.ModTime()

	tctbl := []struct {
		path    string
		mtime   time.Time
		exist   bool
		updated bool
	}{
		{path: "testwatcher/exist", mtime: fimtime, exist: true, updated: true},
		{path: "testwatcher/notexist", exist: false, updated: false},
	}
	for _, tc := range tctbl {
		f := NewFileMonitor(tc.path)
		ok := f.UpdateMtime()
		if ok {
			assert.Equal(t, tc.exist, f.Exist)
			assert.Equal(t, tc.updated, f.Updated)
			assert.Equal(t, tc.mtime, f.Mtime)
		} else {
			assert.Equal(t, tc.exist, f.Exist)
			assert.Equal(t, tc.updated, f.Updated)
		}

		f.ResetUpdate()
		assert.Equal(t, false, f.Updated)
		assert.Equal(t, myinotifyUnknown, f.Event.Op)
	}
}
