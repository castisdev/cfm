package fmfm

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
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

func chtimesfile(dir, filename string) {
	fp := filepath.Join(dir, filename)
	mtime := time.Date(2006, time.February, 1, 3, 4, 5, 0, time.UTC)
	atime := time.Date(2007, time.March, 2, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes(fp, atime, mtime); err != nil {
		log.Fatal(err)
	}
}

func waitFsNotify(w *fsnotify.Watcher, done chan bool, dir, filename string, op fsnotify.Op) {
	path := filepath.Join(dir, filename)
	for {
		select {
		case err, ok := <-w.Errors:
			if !ok {
				log.Println("fsnotify test error: channel closed")
				done <- false
				return
			}
			log.Println("fsnotify test error:", err)
			done <- false
			return
		case e, ok := <-w.Events:
			if !ok {
				log.Println("fsnotify test error: channel closed")
				done <- false
				return
			}
			if e.Op&op == op && e.Name == path {
				done <- true
				return
			}
		case <-time.After(2 * time.Second):
			log.Println("fsnotify test error: timeout")
			done <- false
			return
		}
	}
}

func TestFsNotify() bool {
	dir := "fstest.tmp"
	createdir(dir)
	defer deletefile(dir, "")

	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("fsnotify test error:", err)
		return false
	}
	defer w.Close()
	err = w.Add(dir)
	if err != nil {
		log.Println("fsnotify test error:", err)
		return false
	}
	var r bool
	var rc chan bool
	rc = make(chan bool, 1)
	go waitFsNotify(w, rc, dir, "hello.txt", fsnotify.Create)
	createfile(dir, "hello.txt")
	r = <-rc
	if !r {
		return false
	}
	go waitFsNotify(w, rc, dir, "hello.txt", fsnotify.Write)
	writefile(dir, "hello.txt")
	r = <-rc
	if !r {
		return false
	}
	go waitFsNotify(w, rc, dir, "hello.txt", fsnotify.Chmod)
	chmodfile(dir, "hello.txt")
	r = <-rc
	if !r {
		return false
	}
	go waitFsNotify(w, rc, dir, "hello.txt", fsnotify.Chmod)
	chtimesfile(dir, "hello.txt")
	r = <-rc
	if !r {
		return false
	}
	go waitFsNotify(w, rc, dir, "hello.txt", fsnotify.Remove)
	deletefile(dir, "hello.txt")
	r = <-rc
	if !r {
		return false
	}
	return true
}
