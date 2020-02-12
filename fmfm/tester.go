package fmfm

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/castisdev/cfm/myinotify"
)

func createdir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			log.Fatal(err)
		}
	}
}

func createfile(dir string, filename string) {
	fp := filepath.Join(dir, filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			log.Fatal(err)
		}
	}
	if filename == "" {
		return
	}
	f, err := os.OpenFile(fp, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func writefile(dir string, filename string) {
	fp := filepath.Join(dir, filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			log.Fatal(err)
		}
	}
	f, err := os.OpenFile(fp, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(f, "%s\n", "hello,world")
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
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

func chmodfile(dir, filename string) {
	fp := filepath.Join(dir, filename)
	if err := os.Chmod(fp, 0644); err != nil {
		log.Fatal(err)
	}
}

func renamefile(dir, file, newfile string) {
	oldname := filepath.Join(dir, file)
	newname := filepath.Join(dir, newfile)
	if err := os.Rename(oldname, newname); err != nil {
		log.Fatal(err)
	}
}

func renamedir(dir, newdir string) {
	oldname := dir
	newname := newdir
	if err := os.Rename(oldname, newname); err != nil {
		log.Fatal(err)
	}
}

func chtimesfile(dir, filename string) {
	fp := filepath.Join(dir, filename)
	mtime := time.Now()
	atime := time.Date(2020, time.February, 11, 18, 7, 7, 0, time.UTC)
	if err := os.Chtimes(fp, atime, mtime); err != nil {
		log.Fatal(err)
	}
}

func waitNotify(w *myinotify.Watcher, done chan bool, dir, filename string, op myinotify.Op) {
	path := filepath.Join(dir, filename)
	for {
		select {
		case err, ok := <-w.Errors:
			if !ok {
				log.Println("notify test error: channel closed")
				done <- false
				return
			}
			log.Println("notify test error:", err)
			done <- false
			return
		case e, ok := <-w.Events:
			if !ok {
				log.Println("notify test error: channel closed")
				done <- false
				return
			}
			if e.Op&op == op && e.Name == path {
				done <- true
				return
			}
		case <-time.After(2 * time.Second):
			log.Println("notify test error: timeout")
			done <- false
			return
		}
	}
}

func TestNotify() bool {
	dir := "notifytest.tmp"
	file := "event.file"
	createdir(dir)
	defer deletefile(dir, "")

	w, err := myinotify.NewWatcher()
	if err != nil {
		log.Println("notify test error:", err)
		return false
	}
	defer w.Close()
	err = w.Add(dir)
	if err != nil {
		log.Println("notify test error:", err)
		return false
	}
	var r bool
	var rc chan bool
	rc = make(chan bool, 1)
	go waitNotify(w, rc, dir, file, myinotify.Create)
	createfile(dir, file)
	r = <-rc
	if !r {
		return false
	}
	go waitNotify(w, rc, dir, file, myinotify.Write)
	writefile(dir, file)
	r = <-rc
	if !r {
		return false
	}
	go waitNotify(w, rc, dir, file, myinotify.Chmod)
	chmodfile(dir, file)
	r = <-rc
	if !r {
		return false
	}
	go waitNotify(w, rc, dir, file, myinotify.Chmod)
	chtimesfile(dir, file)
	r = <-rc
	if !r {
		return false
	}
	go waitNotify(w, rc, dir, file, myinotify.Remove)
	deletefile(dir, file)
	r = <-rc
	if !r {
		return false
	}
	return true
}
