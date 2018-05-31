package tasker

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Status is custom type for const
type Status int

// task Status const
const (
	_            = iota
	READY Status = iota
	DONE
	WORKING
	TIMEOUT
)

// MarshalJSON :
func (s Status) MarshalJSON() ([]byte, error) {
	switch s {
	case READY:
		return []byte(`"ready"`), nil
	case DONE:
		return []byte(`"done"`), nil
	case WORKING:
		return []byte(`"working"`), nil
	case TIMEOUT:
		return []byte(`"timeout"`), nil
	default:
		return nil, errors.New("Status.MarshalJSON: unknown value")
	}
}

// UnmarshalJSON :
func (s *Status) UnmarshalJSON(b []byte) error {
	switch string(b) {
	case `"ready"`:
		*s = READY
	case `"done"`:
		*s = DONE
	case `"working"`:
		*s = WORKING
	case `"timeout"`:
		*s = TIMEOUT
	default:
		return fmt.Errorf("unknown Status : (%s)", string(b))
	}

	return nil
}

func (s Status) String() string {
	m := map[Status]string{
		READY:   "ready",
		DONE:    "done",
		WORKING: "working",
		TIMEOUT: "timeout",
	}

	return m[s]
}

// Task is struct for copy task
type Task struct {
	ID       int64  `json:"id,string"`
	Ctime    int64  `json:"ctime"`
	Mtime    int64  `json:"mtime"`
	Status   Status `json:"status"`
	SrcIP    string `json:"src_ip"`
	DstIP    string `json:"dst_ip"`
	FilePath string `json:"file_path"`
	FileName string `json:"file_name"`
	Grade    int32  `json:"grade"`
}

// Tasks is slice of Task struct
type Tasks struct {
	mutex   *sync.Mutex
	TaskMap map[int64]*Task
}

// NewTasks is constructor of Tasks
func NewTasks() *Tasks {
	/*
		for i:=0; i<size; i++ {

		}
		new(Task)
	*/
	tmp := make(map[int64]*Task)
	return &Tasks{&sync.Mutex{}, tmp}
}

// GetTaskList is to get task list as Task slice
func (tasks Tasks) GetTaskList() (tl []Task) {
	tasks.mutex.Lock()
	defer tasks.mutex.Unlock()

	for _, v := range tasks.TaskMap {
		tl = append(tl, *v)
	}

	return
}

// FindTaskByID is to find task with task ID
func (tasks Tasks) FindTaskByID(id int64) (Task, bool) {

	tasks.mutex.Lock()
	defer tasks.mutex.Unlock()

	for _, task := range tasks.TaskMap {
		if task.ID == id {
			return *task, true
		}
	}

	return Task{}, false
}

// FindTaskByFileName is to find task with task ID
func (tasks Tasks) FindTaskByFileName(name string) (Task, bool) {

	tasks.mutex.Lock()
	defer tasks.mutex.Unlock()

	for _, task := range tasks.TaskMap {
		if task.FileName == name {
			return *task, true
		}
	}
	return Task{}, false
}

// UpdateStatus is to change status
func (tasks *Tasks) UpdateStatus(id int64, s Status) error {

	tasks.mutex.Lock()
	defer tasks.mutex.Unlock()

	switch s {

	case READY:
		return fmt.Errorf("invalid request (try to change to %d)", READY)

	case WORKING:
		for _, task := range tasks.TaskMap {
			if task.ID == id {

				if task.Status == WORKING {
					return fmt.Errorf("task already working, (%d), (%s)", task.ID, task.FileName)
				}

				if task.Status == DONE {
					return fmt.Errorf("task already done, (%d), (%s)", task.ID, task.FileName)
				}
				task.Status = s
				task.Mtime = time.Now().Unix()
				return nil
			}
		}
		return fmt.Errorf("(%d) task not found", id)

	case DONE:
		for _, task := range tasks.TaskMap {
			if task.ID == id {
				task.Status = s
				task.Mtime = time.Now().Unix()
				return nil
			}
		}
		return fmt.Errorf("(%d) task not found", id)

	default:
		return fmt.Errorf("invalid request's status (%d)", s)
	}

}

// CreateTask is to create task
func (tasks *Tasks) CreateTask(task *Task) Task {

	task.ID = time.Now().UnixNano()
	task.Ctime = time.Now().Unix()
	task.Mtime = time.Now().Unix()
	task.Status = READY

	tasks.mutex.Lock()
	defer tasks.mutex.Unlock()
	tasks.TaskMap[task.ID] = task

	return *task
}

// DeleteTask is to delete task
func (tasks *Tasks) DeleteTask(id int64) error {

	tasks.mutex.Lock()
	defer tasks.mutex.Unlock()

	_, exists := tasks.TaskMap[id]
	if !exists {
		return fmt.Errorf("could not find Task with id(%d) to delete", id)
	}

	delete(tasks.TaskMap, id)

	return nil

}
