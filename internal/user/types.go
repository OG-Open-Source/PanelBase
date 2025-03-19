package user

// UserRole defines the permission level for a user
type UserRole string

const (
	// RoleRoot is the highest permission level (equivalent to root/admin)
	RoleRoot UserRole = "ROOT"
	// RoleAdmin has high permissions but below root
	RoleAdmin UserRole = "ADMIN"
	// RoleUser has basic permissions
	RoleUser UserRole = "USER"
	// RoleGuest has minimal permissions
	RoleGuest UserRole = "GUEST"
)

// User represents a system user
type User struct {
	ID       int      `json:"id"`
	Username string   `json:"username"`
	Password string   `json:"password,omitempty"` // Hashed password
	Role     UserRole `json:"role"`
	APIKey   string   `json:"api_key,omitempty"` // API key for programmatic access
	Active   bool     `json:"active"`
}

// UserManager defines the interface for user management operations
type UserManager interface {
	GetUser(id int) (*User, error)
	GetUserByUsername(username string) (*User, error)
	GetUserByAPIKey(apiKey string) (*User, error)
	ListUsers() ([]*User, error)
	CreateUser(username, password string, role UserRole) (*User, error)
	UpdateUser(id int, updates map[string]interface{}) error
	DeleteUser(id int) error
	Authenticate(username, password string) (*User, error)
	ValidateAPIKey(apiKey string) (*User, error)
}

// Permission represents an action that can be performed
type Permission string

const (
	PermCreateTask Permission = "CREATE_TASK"
	PermRunTask    Permission = "RUN_TASK"
	PermStopTask   Permission = "STOP_TASK"
	PermDeleteTask Permission = "DELETE_TASK"
	PermViewTask   Permission = "VIEW_TASK"

	PermCreateUser Permission = "CREATE_USER"
	PermUpdateUser Permission = "UPDATE_USER"
	PermDeleteUser Permission = "DELETE_USER"
	PermViewUser   Permission = "VIEW_USER"

	PermSystemConfig Permission = "SYSTEM_CONFIG"
)

// RolePermissions maps roles to their allowed permissions
var RolePermissions = map[UserRole][]Permission{
	RoleRoot: {
		PermCreateTask, PermRunTask, PermStopTask, PermDeleteTask, PermViewTask,
		PermCreateUser, PermUpdateUser, PermDeleteUser, PermViewUser,
		PermSystemConfig,
	},
	RoleAdmin: {
		PermCreateTask, PermRunTask, PermStopTask, PermDeleteTask, PermViewTask,
		PermViewUser,
	},
	RoleUser: {
		PermCreateTask, PermRunTask, PermStopTask, PermViewTask,
	},
	RoleGuest: {
		PermViewTask,
	},
}
