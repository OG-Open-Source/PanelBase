package executor

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
	"github.com/google/uuid"
	"github.com/OG-Open-Source/PanelBase/internal/logger"
)

type Executor struct {
	tasks    map[string]*Task
	mutex    sync.RWMutex
	commands map[string]*exec.Cmd
}

func NewExecutor() *Executor {
	return &Executor{
		tasks:    make(map[string]*Task),
		commands: make(map[string]*exec.Cmd),
	}
}

func (e *Executor) CreateTask(name string, commands []Command, workDir string) (*Task, error) {
	task := &Task{
		ID:       uuid.New().String(),
		Name:     name,
		Commands: commands,
		WorkDir:  workDir,
		Status:   StatusPending,
	}

	e.mutex.Lock()
	e.tasks[task.ID] = task
	e.mutex.Unlock()

	logger.Info(fmt.Sprintf("Created task: %s (%s) with %d commands", task.Name, task.ID, len(commands)))
	return task, nil
}

func (e *Executor) StartTask(taskID string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	task, exists := e.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.Status == StatusRunning {
		return fmt.Errorf("task already running: %s", taskID)
	}

	task.StartTime = time.Now().Unix()
	task.Status = StatusRunning
	var outputs []string

	// 依序執行每個命令
	for i, cmd := range task.Commands {
		logger.Info(fmt.Sprintf("Executing command %d/%d: %s %v", i+1, len(task.Commands), cmd.Command, cmd.Args))
		
		command := exec.Command(cmd.Command, cmd.Args...)
		command.Dir = task.WorkDir

		output, err := command.CombinedOutput()
		outputs = append(outputs, fmt.Sprintf("$ %s %s\n%s", 
			cmd.Command, 
			strings.Join(cmd.Args, " "), 
			string(output)))

		if err != nil {
			task.Status = StatusFailed
			task.Error = fmt.Sprintf("Command %d failed: %v", i+1, err)
			task.EndTime = time.Now().Unix()
			task.Output = strings.Join(outputs, "\n---\n")
			logger.Error(fmt.Sprintf("Task failed: %s (%s): %v", task.Name, task.ID, err))
			return err
		}
	}

	task.Status = StatusCompleted
	task.EndTime = time.Now().Unix()
	task.Output = strings.Join(outputs, "\n---\n")
	task.ExitCode = 0

	logger.Info(fmt.Sprintf("Task completed: %s (%s)", task.Name, task.ID))
	return nil
}

func (e *Executor) GetTaskStatus(taskID string) (*Task, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	task, exists := e.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

func (e *Executor) StopTask(taskID string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	task, exists := e.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.Status != StatusRunning {
		return fmt.Errorf("task is not running: %s", taskID)
	}

	cmd, exists := e.commands[taskID]
	if exists && cmd.Process != nil {
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop task: %v", err)
		}
		delete(e.commands, taskID)
	}

	task.Status = StatusFailed
	task.EndTime = time.Now().Unix()
	task.Error = "Task stopped by user"

	return nil
}

func (e *Executor) ListTasks() ([]*Task, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	tasks := make([]*Task, 0, len(e.tasks))
	for _, task := range e.tasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// ... 其他方法實現 