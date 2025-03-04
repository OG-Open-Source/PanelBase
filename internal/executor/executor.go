package executor

import (
	"fmt"
	"os/exec"
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

func (e *Executor) CreateTask(name, command string, args []string, workDir string) (*Task, error) {
	task := &Task{
		ID:      uuid.New().String(),
		Name:    name,
		Command: command,
		Args:    args,
		WorkDir: workDir,
		Status:  StatusPending,
	}

	e.mutex.Lock()
	e.tasks[task.ID] = task
	e.mutex.Unlock()

	logger.Info(fmt.Sprintf("Created task: %s (%s)", task.Name, task.ID))
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

	// 設置任務開始時間和狀態
	task.StartTime = time.Now().Unix()
	task.Status = StatusRunning

	cmd := exec.Command(task.Command, task.Args...)
	cmd.Dir = task.WorkDir

	// 捕獲輸出
	output, err := cmd.CombinedOutput()
	if err != nil {
		task.Status = StatusFailed
		task.Error = err.Error()
		task.EndTime = time.Now().Unix()
		logger.Error(fmt.Sprintf("Task failed: %s (%s): %v", task.Name, task.ID, err))
		return err
	}

	task.Output = string(output)
	task.Status = StatusCompleted
	task.EndTime = time.Now().Unix()
	task.ExitCode = 0

	logger.Info(fmt.Sprintf("Task completed: %s (%s)", task.Name, task.ID))
	logger.Info(fmt.Sprintf("Output: %s", task.Output))

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