package main

import (
	"fmt"
)

type Node struct {
	ID   int
	Name string
	// Add other attributes and methods related to the node
}

type NodeTaskHandler struct {
	Node *Node
}

func (h *NodeTaskHandler) Handle() error {
	// Process the node
	ProcessNode(h.Node)
	return nil
}

func ProcessNode(node *Node) {
	fmt.Printf("Processing Node %d - %s\n", node.ID, node.Name)
	// Your node processing logic here
}

func main() {
	nodes := []*Node{
		{ID: 1, Name: "Node 1"},
		{ID: 2, Name: "Node 2"},
		{ID: 3, Name: "Node 3"},
		{ID: 4, Name: "Node 4"},
		{ID: 5, Name: "Node 5"},
	}

	// Convert each node to a task
	tasks := make([]*Task, len(nodes))
	for i, node := range nodes {
		handler := &NodeTaskHandler{Node: node}
		tasks[i] = NewTask(fmt.Sprintf("NodeTask_%d", node.ID), handler)
	}

	// Assume you have a list of lines that connect nodes
	lines := []struct {
		FromNodeID int
		ToNodeID   int
	}{
		{FromNodeID: 1, ToNodeID: 2},
		{FromNodeID: 1, ToNodeID: 3},
		{FromNodeID: 2, ToNodeID: 4},
		{FromNodeID: 3, ToNodeID: 4},
		{FromNodeID: 4, ToNodeID: 5},
	}

	// Build dependency relationship based on lines
	for _, line := range lines {
		fromTask := tasks[line.FromNodeID-1]
		toTask := tasks[line.ToNodeID-1]
		toTask.AddDependency(fromTask)
	}

	// Create and start task listener
	listener := NewTaskListener(tasks)
	go listener.PrintStatus()

	// Start executing tasks
	listener.Start()
	listener.Wait()
}
