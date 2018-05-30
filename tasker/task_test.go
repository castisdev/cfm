package tasker

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTasks_New(t *testing.T) {

	tasks := NewTasks()
	assert.Equal(t, 0, len(tasks.TaskMap))
}

func TestTasks_FindTaskByID(t *testing.T) {
	tasks := NewTasks()

	t1 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg"})
	t2 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg"})
	t3 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg"})
	tl := []Task{t1, t2, t3}

	for _, task := range tl {

		findTask, exists := tasks.FindTaskByID(task.ID)
		assert.Equal(t, true, exists)
		assert.Equal(t, true, reflect.DeepEqual(task.ID, findTask.ID))
	}
}

func TestTasks_CreateTask(t *testing.T) {

	tasks := NewTasks()
	tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg"})
	tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg"})
	tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg"})

	assert.Equal(t, 3, len(tasks.TaskMap))

}

func TestTasks_CreateTask_Mutex(t *testing.T) {

	tasks := NewTasks()

	wg := new(sync.WaitGroup)

	wg.Add(1)
	go func() {

		for i := 1; i < 1001; i++ {
			tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg"})

		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for i := 1; i < 1001; i++ {
			tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg"})
		}
		wg.Done()
	}()

	wg.Wait()
	// 2개의 GoRoutin 이 각각 1000 개씩 task 생성
	assert.Equal(t, 2000, len(tasks.TaskMap))

}

func TestTasks_DeleteTask(t *testing.T) {

	tasks := NewTasks()

	t1 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg"})
	t2 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg"})
	t3 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg"})

	tasks.DeleteTask(t1.ID)
	assert.Equal(t, 2, len(tasks.TaskMap))

	// t1 은 이미 삭제했으므로 존재해선 안됨
	for _, task := range tasks.TaskMap {
		assert.NotEqual(t, task.ID, t1.ID)
	}

	tasks.DeleteTask(t2.ID)
	assert.Equal(t, 1, len(tasks.TaskMap))

	tasks.DeleteTask(t3.ID)
	assert.Equal(t, 0, len(tasks.TaskMap))
}

func TestTasks_FindTaskByFileName(t *testing.T) {
	tasks := NewTasks()

	t1 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg", FileName: "A.mpg"})
	t2 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg", FileName: "B.mpg"})
	t3 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg", FileName: "C.mpg"})

	tl := []Task{t1, t2, t3}

	for _, task := range tl {
		_, exists := tasks.FindTaskByFileName(task.FileName)
		assert.Equal(t, true, exists)
	}

	_, exists := tasks.FindTaskByFileName("D.mpg")
	assert.Equal(t, false, exists)
}

func TestTasks_UpdateStatus(t *testing.T) {

	tasks := NewTasks()

	// task 를 복사해서 리턴한다. 즉 값이 복사된다.
	t1 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg", FileName: "A.mpg"})

	assert.Nil(t, tasks.UpdateStatus(t1.ID, WORKING))

	// t1 은 값이 복사된 별개의 객체이기 때문에
	// t1.Status 로 비교하면 안된다.
	for _, task := range tasks.TaskMap {
		if task.ID == t1.ID {
			assert.Equal(t, WORKING, task.Status)
		}
	}

}
