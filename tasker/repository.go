package tasker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cilog"
	"github.com/syndtr/goleveldb/leveldb"
)

// Repository
// leveldb의 단순 wrapper이다.
type Repository struct {
	where  string
	logger common.MLogger
	db     *leveldb.DB
	isOpen bool
}

func newRepository() *Repository {
	r := &Repository{
		where: ".repository/tasks.db",
		logger: common.MLogger{
			Logger: cilog.StdLogger(),
			Mod:    "repository"},
		isOpen: false,
	}
	return r
}

func (r *Repository) remove() error {
	if r.isOpen {
		r.close()
	}
	return os.RemoveAll(filepath.Dir(r.where))
}

func (r *Repository) open() error {
	db, err := leveldb.OpenFile(r.where, nil)
	if err != nil {
		r.logger.Errorf("failed to open tasks repository")
		return err
	}
	r.db = db
	r.isOpen = true
	return nil
}

func (r *Repository) close() error {
	if !r.isOpen {
		return nil
	}
	err := r.db.Close()
	if err != nil {
		r.logger.Errorf("failed to close tasks repository")
		return err
	}
	r.isOpen = false
	return nil
}

func (r *Repository) saveTask(t *Task) error {
	if !r.isOpen {
		if err := r.open(); err != nil {
			return err
		}
	}
	tv, err := json.Marshal(t)
	err = r.db.Put([]byte(strconv.FormatInt(t.ID, 10)), []byte(tv), nil)
	if err != nil {
		r.logger.Errorf("[%d] failed to save task", t.ID)
		return err
	}
	return nil
}

func (r *Repository) deleteTask(id int64) error {
	if !r.isOpen {
		if err := r.open(); err != nil {
			return err
		}
	}
	err := r.db.Delete([]byte(strconv.FormatInt(id, 10)), nil)
	if err != nil {
		r.logger.Errorf("[%d] failed to delete task", id)
		return err
	}
	return nil
}

func (r *Repository) loadTasks() ([]Task, error) {
	tasks := make([]Task, 0)
	if !r.isOpen {
		if err := r.open(); err != nil {
			return tasks, err
		}
	}

	iter := r.db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		value := iter.Value()
		t := Task{}
		if err := json.Unmarshal(value, &t); err != nil {
			r.logger.Errorf("failed to load task, %s", value)
			return tasks, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
