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

func TestTasks_Create_ManyTask(t *testing.T) {
	tasks := NewTasks()
	defer tasks.DeleteAllTask()

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		for i := 0; i < 100; i++ {
			tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg"})
		}
		wg.Done()
	}()

	wg.Wait()
	// 1개의 GoRoutine이 100개 task 생성
	assert.Equal(t, 100, len(tasks.TaskMap))

	// repository에서 task load
	// tasks.TaskMap 에 이미 100개가 들어있고,
	// 같은 task들이 repository에도 100개가 들어있는 상태여서
	// repository에서 LoadTasks 해도 여전히 100개임
	tasks.LoadTasks()
	assert.Equal(t, 100, len(tasks.TaskMap))

	// 정보를 모두 지우면, repository에서도 지워지기 때문에,
	//  다시 load 해도 정보는 없음
	tasks.DeleteAllTask()
	tasks.LoadTasks()
	assert.Equal(t, 0, len(tasks.TaskMap))
}

func TestTasks_TwoCreator_Create_ManyTask(t *testing.T) {
	tasks := NewTasks()
	defer tasks.DeleteAllTask()

	wg := new(sync.WaitGroup)

	wg.Add(1)
	go func() {

		for i := 0; i < 10000; i++ {
			tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg"})

		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for i := 0; i < 10000; i++ {
			tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg"})
		}
		wg.Done()
	}()

	wg.Wait()
	// 2개의 GoRoutine이 각각 10000 개씩 task 생성
	assert.Equal(t, 20000, len(tasks.TaskMap))
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

func TestTasks_GetTaskList(t *testing.T) {
	tasks := NewTasks()

	t1 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/A.mpg", FileName: "A.mpg"})
	t2 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/B.mpg", FileName: "B.mpg"})
	t3 := tasks.CreateTask(&Task{SrcIP: "127.0.0.1", FilePath: "/data2/C.mpg", FileName: "C.mpg"})

	// sort 된 list 를 반환함
	tl := tasks.GetTaskList()

	assert.Equal(t, t1.ID, tl[0].ID)
	assert.Equal(t, t2.ID, tl[1].ID)
	assert.Equal(t, t3.ID, tl[2].ID)
}
