package tasker

import (
	"errors"
	"fmt"
	"sort"
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

type TaskTime int64

// Task is struct for 배포 task
//
// ID : unique key값
//
// CTime : created time, 사용하는 곳 없음
//
// MTime : modified time, 배포 task Timeout 삭제 조건으로 사용
//
// Status : 배포 상태
//
// SrcIP : source 서버 IP
//
// DstIP : dest 서버 IP, 사용하는 곳 없음(DstAddr과 중복됨)
//
// FilePath : 배포 파일 path, source 서버 상의 파일 위치
//
// FileName : 배포 파일 이름
//
// Grade : 파일 등급, 사용하는 곳 없음
//
// CopySpeed : 배포 속도(bps)
//
// SrcAddr : Source 서버 IP, Port
//
// DstAddr : Destination 서버 IP, Port
type Task struct {
	ID        int64    `json:"id,string"`
	Ctime     TaskTime `json:"ctime"`
	Mtime     TaskTime `json:"mtime"`
	Status    Status   `json:"status"`
	SrcIP     string   `json:"src_ip"`
	DstIP     string   `json:"dst_ip"`
	FilePath  string   `json:"file_path"`
	FileName  string   `json:"file_name"`
	Grade     int32    `json:"grade"`
	CopySpeed string   `json:"copy_speed"`
	SrcAddr   string   `json:"src_addr"`
	DstAddr   string   `json:"dst_addr"`
}

// NewTaskFrom : param으로 받은 task로부터 cloning한 새로운 task반환
func NewTaskFrom(t Task) *Task {
	return &Task{
		ID:        t.ID,
		Ctime:     t.Ctime,
		Mtime:     t.Mtime,
		Status:    t.Status,
		SrcIP:     t.SrcIP,
		DstIP:     t.DstIP,
		FilePath:  t.FilePath,
		FileName:  t.FileName,
		Grade:     t.Grade,
		CopySpeed: t.CopySpeed,
		SrcAddr:   t.SrcAddr,
		DstAddr:   t.DstAddr,
	}
}

func (t TaskTime) String() string {
	return time.Unix(int64(t), 0).Format(time.RFC3339)
}

// String : task to string
func (task Task) String() string {
	t := fmt.Sprintf(
		"id(%d), status(%s), grade(%d), filePath(%s), fileName(%s)"+
			", srcAddr(%s), dstAddr(%s)"+
			", mtime(%s), copySpeed(%s)bps, srcIP(%s), ctime(%s), dstIP(%s)",
		task.ID, task.Status, task.Grade, task.FilePath, task.FileName,
		task.SrcAddr, task.DstAddr, task.Mtime, task.CopySpeed, task.SrcIP,
		task.Ctime, task.DstIP,
	)
	return t
}

// Tasks is slice of Task struct
//
// map은 read만 있을 때는 thread-safe 하지만,
// update가 있을 때는 그렇지 않음
//
// https://golang.org/doc/faq#atomic_maps
//
// https://blog.golang.org/go-maps-in-action
type Tasks struct {
	mutex      *sync.RWMutex
	TaskMap    map[int64]*Task
	repository *Repository
}

// NewTasks is constructor of Tasks
func NewTasks() *Tasks {
	return &Tasks{
		&sync.RWMutex{},
		make(map[int64]*Task),
		newRepository()}
}

// GetTaskList :
// map으로 된 task 정보에서 task list를 반환
//
// lock 을 사용하고, 작은 taskID 순으로 sort해서 반환
//
// create nano time으로 id를 만들기 때문에 먼저 만들어진 task 가 먼저 나오게 됨
func (tasks Tasks) GetTaskList() []Task {
	tasks.mutex.RLock()
	defer tasks.mutex.RUnlock()

	tl := make([]Task, 0)
	for _, v := range tasks.TaskMap {
		tl = append(tl, *v)
	}
	sort.Slice(tl, func(i, j int) bool {
		return tl[i].ID < tl[j].ID
	})

	return tl
}

// FindTaskByID is to find task with task ID
func (tasks Tasks) FindTaskByID(id int64) (Task, bool) {

	tasks.mutex.RLock()
	defer tasks.mutex.RUnlock()

	for _, task := range tasks.TaskMap {
		if task.ID == id {
			return *task, true
		}
	}

	return Task{}, false
}

// FindTaskByFileName is to find task with task ID
func (tasks Tasks) FindTaskByFileName(name string) (Task, bool) {

	tasks.mutex.RLock()
	defer tasks.mutex.RUnlock()

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

	task, found := tasks.TaskMap[id]
	if !found {
		return fmt.Errorf("not found,task(%d)", id)
	}

	switch s {

	case READY:
		// fail : READY, WORKING, TIMEOUT, DONE -> READY
		return fmt.Errorf("invalid request, try to change to status(%s)", READY)

	case WORKING:
		// fail : TIMEOUT, DONE -> WORKING
		if task.Status == DONE || task.Status == TIMEOUT {
			return fmt.Errorf("invalid request, task already done or tiemout,task(%d),"+
				"filename(%s)", task.ID, task.FileName)
		}
		// success : READY, WORKING-> WORKING
		task.Status = s
		task.Mtime = TaskTime(time.Now().Unix())
		tasks.repository.saveTask(task)
		return nil

	case DONE:
		// TIMEOUT -> DONE 도 ok
		// success : READY, WORKING, DONE, TIMEOUT -> DONE
		task.Status = s
		task.Mtime = TaskTime(time.Now().Unix())
		tasks.repository.saveTask(task)
		return nil

	case TIMEOUT:
		// DONE -> TIMEOUT 도 ok
		// success : READY, WORKING, DONE, TIMEOUT -> TIMEOUT
		task.Status = s
		task.Mtime = TaskTime(time.Now().Unix())
		tasks.repository.saveTask(task)
		return nil

	default:
		return fmt.Errorf("invalid request,status(unkown)")
	}

}

// LoadTasks :
// repository 에서 task list load
//
// load 중에 repository 에러가 나는 경우,
//
// 잘못된 repository라고 repostiroy는 초기화함
func (tasks *Tasks) LoadTasks() {
	tasks.mutex.Lock()
	defer tasks.mutex.Unlock()
	tl, err := tasks.repository.loadTasks()
	if err != nil {
		tasks.repository.remove()
		return
	}
	for _, t := range tl {
		tasks.TaskMap[t.ID] = NewTaskFrom(t)
	}
}

// CreateTask is to create task
func (tasks *Tasks) CreateTask(task *Task) Task {

	task.ID = time.Now().UnixNano()
	task.Ctime = TaskTime(time.Now().Unix())
	task.Mtime = TaskTime(time.Now().Unix())
	task.Status = READY

	tasks.mutex.Lock()
	defer tasks.mutex.Unlock()
	tasks.TaskMap[task.ID] = task

	tasks.repository.saveTask(task)
	return *task
}

// DeleteTask is to delete task
func (tasks *Tasks) DeleteTask(id int64) error {

	tasks.mutex.Lock()
	defer tasks.mutex.Unlock()

	_, exists := tasks.TaskMap[id]
	if !exists {
		return fmt.Errorf("not found, task(%d) to delete", id)
	}

	delete(tasks.TaskMap, id)
	tasks.repository.deleteTask(id)
	return nil
}

// DeleteTasks is to delete tasks
// id 가 없으면 그냥 다음 id로 넘어감
// 성공한 개수가 넘어감
func (tasks *Tasks) DeleteTasks(ids []int64) int {
	tasks.mutex.Lock()
	defer tasks.mutex.Unlock()

	var cnt int = 0
	for _, id := range ids {
		_, exists := tasks.TaskMap[id]
		if exists {
			delete(tasks.TaskMap, id)
			tasks.repository.deleteTask(id)
			cnt++
		}
	}
	return cnt
}

// DeleteAllTask
//
// 내부 map, repository를 모두 지우고, 새롭게 할당
func (tasks *Tasks) DeleteAllTask() {
	tasks.mutex.Lock()
	defer tasks.mutex.Unlock()

	tasks.repository.remove()
	tasks.TaskMap = make(map[int64]*Task)
	tasks.repository = newRepository()
}

// Release
// 내부 map, repository를 모두 지우기
func (tasks *Tasks) Release() {
	tasks.mutex.Lock()
	defer tasks.mutex.Unlock()
	tasks.repository.remove()
	tasks.TaskMap = make(map[int64]*Task)
}
