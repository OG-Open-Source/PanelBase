package executor

// TaskStatus 表示任務狀態
type TaskStatus string

const (
	StatusPending   TaskStatus = "PENDING"
	StatusRunning   TaskStatus = "RUNNING"
	StatusCompleted TaskStatus = "COMPLETED"
	StatusFailed    TaskStatus = "FAILED"
)

// Command 表示單個命令
type Command struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// Task 表示一個執行任務
type Task struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Commands    []Command  `json:"commands"`    // 改為命令數組
	WorkDir     string     `json:"work_dir"`
	Status      TaskStatus `json:"status"`
	Output      string     `json:"output"`
	Error       string     `json:"error"`
	StartTime   int64      `json:"start_time"`
	EndTime     int64      `json:"end_time"`
	ExitCode    int        `json:"exit_code"`
}

// TaskManager 接口定義任務管理器的行為
type TaskManager interface {
	CreateTask(name, command string, args []string, workDir string) (*Task, error)
	StartTask(taskID string) error
	StopTask(taskID string) error
	GetTaskStatus(taskID string) (*Task, error)
	ListTasks() ([]*Task, error)
}