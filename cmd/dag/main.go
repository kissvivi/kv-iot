package main

import (
	"fmt"
	"sync"
	"time"
)

type TaskState int32

const (
	TaskStateInit    = 1
	TaskStateRunning = 2
	TaskStateDone    = 3
)

type TaskHandler interface {
	Handle() error
}

type Task struct {
	name        string
	dependences []*Task
	state       TaskState
	handler     TaskHandler
	doneChannel chan struct{} // Channel to notify task completion
	mutex       sync.Mutex    // Mutex to protect access to task state
	doneSignal  chan bool     // Channel to receive external completion signal
}

func NewTask(n string, h TaskHandler) *Task {
	return &Task{
		name:        n,
		state:       TaskStateInit,
		handler:     h,
		doneChannel: make(chan struct{}),
		doneSignal:  make(chan bool),
	}
}

func (t *Task) AddDependency(dependency *Task) {
	t.dependences = append(t.dependences, dependency)
}

func (t *Task) Execute() {
	if t.state == TaskStateDone {
		return
	}

	for _, dep := range t.dependences {
		<-dep.doneChannel // Wait for the dependent task to complete
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.state = TaskStateRunning // Set the task state to "Running"
	fmt.Printf("Task '%s' started at %v\n", t.name, time.Now())

	//go func() {
	//	err := t.handler.Handle()
	//	if err != nil {
	//		fmt.Printf("Task '%s' failed: %v\n", t.name, err)
	//	}
	//
	//	time.Sleep(3 * time.Second)
	//
	//	t.mutex.Lock()
	//	t.state = TaskStateDone
	//	t.mutex.Unlock()
	//
	//	fmt.Printf("Task '%s' completed at %v\n", t.name, time.Now())
	//
	//	close(t.doneChannel) // Close the channel to notify completion
	//	t.doneSignal <- true // Signal completion to external notification
	//}()

	select {
	case <-t.doneSignal: // Wait for external completion signal
	case <-time.After(time.Second * 10): // Timeout after 10 seconds (you can adjust the timeout as needed)
		fmt.Printf("Task '%s' is still running after 10 seconds\n", t.name)
	}
}

type PTaskHandler struct {
	Msg string
}

func (p *PTaskHandler) Handle() error {
	fmt.Println(p.Msg)
	return nil
}

type TaskListener struct {
	tasks []*Task
}

func NewTaskListener(tasks []*Task) *TaskListener {
	return &TaskListener{
		tasks: tasks,
	}
}

func (tl *TaskListener) Start() {
	for _, task := range tl.tasks {
		go func(t *Task) {
			t.Execute()
		}(task)
	}
}

func (tl *TaskListener) Wait() {
	for _, task := range tl.tasks {
		<-task.doneChannel // Wait for each task to complete
	}
}

func (tl *TaskListener) PrintStatus() {
	for {
		allCompleted := true
		for _, task := range tl.tasks {
			task.mutex.Lock()
			state := task.state
			task.mutex.Unlock()

			if state != TaskStateDone {
				allCompleted = false
			}
			// TODO 状态入库 或者缓存
			fmt.Printf("Task '%s' state: %s\n", task.name, taskStateToString(state))
		}

		if allCompleted {
			break
		}

		time.Sleep(500 * time.Millisecond) // Print status every 500ms
	}
}

func taskStateToString(state TaskState) string {
	switch state {
	case TaskStateInit:
		return "Init"
	case TaskStateRunning:
		return "Running"
	case TaskStateDone:
		return "Done"
	default:
		return "Unknown"
	}
}

//func main() {
//	task1 := NewTask("1", &PTaskHandler{Msg: "1"})
//	task2 := NewTask("2", &PTaskHandler{Msg: "2"})
//	task3 := NewTask("3", &PTaskHandler{Msg: "3"})
//	task4 := NewTask("4", &PTaskHandler{Msg: "4"})
//	task5 := NewTask("5", &PTaskHandler{Msg: "5"})
//
//	task2.AddDependency(task1)
//	task3.AddDependency(task2)
//	task4.AddDependency(task3)
//	task5.AddDependency(task4)
//
//	tasks := []*Task{task1, task2, task3, task4, task5}
//
//	listener := NewTaskListener(tasks)
//	go listener.PrintStatus() // Start printing task status in a separate goroutine
//	listener.Start()
//	listener.Wait()
//}
