package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
)

// TaskRunner is a generic interface to abstract any task or process you want to run.
type TaskRunner[T any] interface {
	RunTask(task T) error
}

// CommandRunner implements TaskRunner for command-line tasks.
type CommandRunner struct{}

// RunTask runs a provided command-line task.
func (c *CommandRunner) RunTask(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GenericServer struct to hold a generic TaskRunner that can run various types of tasks.
type GenericServer[T any] struct {
	Task TaskRunner[T]
}

// NewGenericServer creates a new generic server with the provided task runner.
func NewGenericServer[T any](taskRunner TaskRunner[T]) *GenericServer[T] {
	return &GenericServer[T]{Task: taskRunner}
}

// Start runs the provided task using the task runner.
func (s *GenericServer[T]) Start(task T, wg *sync.WaitGroup) {
	defer wg.Done()
	if err := s.Task.RunTask(task); err != nil {
		log.Fatalf("Failed to run task: %v", err)
	}
}

func main() {
	// Set up a wait group to manage multiple tasks (if you expand the system)
	var wg sync.WaitGroup

	fmt.Println("Root-level server is up and running...")

	// Create a CommandRunner (for running command-line processes)
	commandRunner := &CommandRunner{}

	// Create a generic server that uses the CommandRunner
	server := NewGenericServer(commandRunner)

	// Define the command to run cmd/server/main.go
	cmd := exec.Command("go", "run", "cmd/server/main.go")

	// Add one to the wait group for the server task
	wg.Add(1)

	// Start the server (cmd/server/main.go) in a goroutine
	go server.Start(cmd, &wg)

	// Wait for all tasks to complete (if you add more tasks in the future, this will handle it)
	wg.Wait()

	fmt.Println("Root-level server has completed all tasks.")
}