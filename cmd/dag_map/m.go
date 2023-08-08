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

type DAGTask struct {
	name         string
	state        TaskState
	handler      TaskHandler
	doneChannel  chan struct{}       // Channel to notify task completion
	mutex        sync.Mutex          // Mutex to protect access to task state
	doneSignal   chan bool           // Channel to receive external completion signal
	dependencies map[string]*DAGTask // Map to store dependencies of the task
	visited      bool                // Flag to track visited state for DFS
	recStack     bool                // Flag to track recursion stack for cycle detection
}

func NewDAGTask(n string, h TaskHandler) *DAGTask {
	return &DAGTask{
		name:         n,
		state:        TaskStateInit,
		handler:      h,
		doneChannel:  make(chan struct{}),
		doneSignal:   make(chan bool),
		dependencies: make(map[string]*DAGTask),
	}
}

func (t *DAGTask) AddDependency(dependency *DAGTask) {
	t.dependencies[dependency.name] = dependency
}

func (t *DAGTask) Execute() {
	if t.state == TaskStateDone {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.state = TaskStateRunning // Set the task state to "Running"
	fmt.Printf("Task '%s' started at %v\n", t.name, time.Now())

	err := t.handler.Handle()
	if err != nil {
		fmt.Printf("Task '%s' failed: %v\n", t.name, err)
	}

	time.Sleep(3 * time.Second)

	t.mutex.Lock()
	t.state = TaskStateDone
	t.mutex.Unlock()

	fmt.Printf("Task '%s' completed at %v\n", t.name, time.Now())

	close(t.doneChannel) // Close the channel to notify completion
	t.doneSignal <- true // Signal completion to external notification
}

type PTaskHandler struct {
	Msg string
}

func (p *PTaskHandler) Handle() error {
	fmt.Println(p.Msg)
	return nil
}

type DAGTaskScheduler struct {
	tasks map[string]*DAGTask // Map to store tasks in the scheduler
}

func NewDAGTaskScheduler() *DAGTaskScheduler {
	return &DAGTaskScheduler{
		tasks: make(map[string]*DAGTask),
	}
}

func (ds *DAGTaskScheduler) AddTask(task *DAGTask) {
	ds.tasks[task.name] = task
}

func (ds *DAGTaskScheduler) Execute() {
	for _, task := range ds.tasks {
		if !task.visited {
			if ds.isCyclic(task) {
				fmt.Printf("Cycle detected in task dependencies: %s\n", task.name)
				return
			}
			ds.executeDFS(task)
		}
	}
}

func (ds *DAGTaskScheduler) isCyclic(task *DAGTask) bool {
	if task.recStack {
		return true
	}

	if !task.visited {
		task.recStack = true
		for _, dep := range task.dependencies {
			if ds.isCyclic(dep) {
				return true
			}
		}
		task.recStack = false
		task.visited = true
	}

	return false
}

func (ds *DAGTaskScheduler) executeDFS(task *DAGTask) {
	task.visited = true
	for _, dep := range task.dependencies {
		if !dep.visited {
			ds.executeDFS(dep)
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func(t *DAGTask) {
		t.Execute()
		wg.Done()
	}(task)

	wg.Wait()
}

func main() {
	scheduler := NewDAGTaskScheduler()

	task1 := NewDAGTask("1", &PTaskHandler{Msg: "1"})
	task2 := NewDAGTask("2", &PTaskHandler{Msg: "2"})
	task3 := NewDAGTask("3", &PTaskHandler{Msg: "3"})
	task4 := NewDAGTask("4", &PTaskHandler{Msg: "4"})
	task5 := NewDAGTask("5", &PTaskHandler{Msg: "5"})

	task1.AddDependency(task2)
	task1.AddDependency(task3)
	task2.AddDependency(task4)
	task3.AddDependency(task4)
	task4.AddDependency(task5)

	scheduler.AddTask(task1)
	scheduler.AddTask(task2)
	scheduler.AddTask(task3)
	scheduler.AddTask(task4)
	scheduler.AddTask(task5)

	scheduler.Execute()
}
