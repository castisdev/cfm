package tasker

import (
	"encoding/json"
	"strconv"

	"github.com/castisdev/cfm/common"
	"github.com/castisdev/cilog"
	"github.com/syndtr/goleveldb/leveldb"
)

type Repository struct {
	where  string
	logger common.MLogger
}

func newRepository() *Repository {
	r := &Repository{
		where: ".repository/tasks.db",
		logger: common.MLogger{
			Logger: cilog.StdLogger(),
			Mod:    "repository"},
	}
	return r
}

func (r *Repository) saveTask(t *Task) {
	db, err := leveldb.OpenFile(r.where, nil)
	if err != nil {
		r.logger.Errorf("fail to open tasks repository")
		return
	}
	defer db.Close()

	tv, err := json.Marshal(t)
	err = db.Put([]byte(strconv.FormatInt(t.ID, 10)), []byte(tv), nil)
	if err != nil {
		r.logger.Errorf("[%d] fail to save task", t.ID)
	}
}

func (r *Repository) deleteTask(id int64) {
	db, err := leveldb.OpenFile(r.where, nil)
	if err != nil {
		r.logger.Errorf("fail to open tasks repository")
		return
	}
	defer db.Close()

	err = db.Delete([]byte(strconv.FormatInt(id, 10)), nil)
	if err != nil {
		r.logger.Errorf("[%d] fail to delete task", id)
	}
}

func (r *Repository) loadTasks() []Task {
	tasks := make([]Task, 0)
	db, err := leveldb.OpenFile(r.where, nil)
	if err != nil {
		r.logger.Errorf("fail to open tasks repository")
		return tasks
	}
	defer db.Close()

	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		value := iter.Value()
		t := Task{}
		if err := json.Unmarshal(value, &t); err != nil {
			r.logger.Errorf("fail to load task, %s", value)
			continue
		}
		tasks = append(tasks, t)
	}

	return tasks
}
