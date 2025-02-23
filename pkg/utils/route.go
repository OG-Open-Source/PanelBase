package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Route struct {
	Path     string   `json:"path"`
	Command  string   `json:"command"`
	Args     []string `json:"args"`
	Method   string   `json:"method"`
	Metadata CommandMetadata
}

type CommandMetadata struct {
	Name         string   `json:"name"`
	PkgManager   []string `json:"pkg_manager"`
	Dependencies []string `json:"dependencies"`
	Author       string   `json:"author"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
}

type RouteManager struct {
	routes map[string]*Route
}

func NewRouteManager() *RouteManager {
	return &RouteManager{
		routes: make(map[string]*Route),
	}
}

func (rm *RouteManager) LoadRoutes(configPath string) error {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		Log(ERROR, "Failed to read routes config: %v", err)
		return err
	}

	var routes map[string]*Route
	if err := json.Unmarshal(data, &routes); err != nil {
		Log(ERROR, "Failed to parse routes config: %v", err)
		return err
	}

	rm.routes = routes
	Log(INFO, "Routes loaded successfully")
	return nil
}

func (rm *RouteManager) ExecuteCommand(route *Route, args map[string]string) (string, error) {
	// Create temp directory for command execution
	tempDir, err := ioutil.TempDir("", "panelbase-cmd-")
	if err != nil {
		Log(ERROR, "Failed to create temp directory: %v", err)
		return "", err
	}
	defer os.RemoveAll(tempDir)

	// Copy command file to temp directory
	cmdPath := filepath.Join("internal/commands", route.Command)
	tempCmdPath := filepath.Join(tempDir, route.Command)
	if err := copyFile(cmdPath, tempCmdPath); err != nil {
		Log(ERROR, "Failed to copy command file: %v", err)
		return "", err
	}

	// Parse command metadata
	if err := rm.parseCommandMetadata(route, tempCmdPath); err != nil {
		Log(ERROR, "Failed to parse command metadata: %v", err)
		return "", err
	}

	// Replace arguments
	processedArgs := make([]string, len(route.Args))
	for i, arg := range route.Args {
		processedArgs[i] = replaceArgs(arg, args)
	}

	// Execute command
	cmd := exec.Command(tempCmdPath, processedArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		Log(ERROR, "Command execution failed: %v", err)
		return "", err
	}

	Log(INFO, "Command executed successfully: %s", route.Command)
	return string(output), nil
}

func (rm *RouteManager) GetRoute(path string) *Route {
	return rm.routes[path]
}

// Helper functions
func copyFile(src, dst string) error {
	input, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, input, 0755)
}

func replaceArgs(template string, args map[string]string) string {
	result := template
	for key, value := range args {
		result = strings.ReplaceAll(result, fmt.Sprintf("*#ARG_%s#*", key), value)
	}
	return result
}

func (rm *RouteManager) parseCommandMetadata(route *Route, cmdPath string) error {
	file, err := os.Open(cmdPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	metadata := CommandMetadata{}
	required := map[string]bool{
		"@commands":     false,
		"@pkg_manager":  false,
		"@author":      false,
		"@version":     false,
		"@description": false,
	}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "//") {
			break
		}

		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "#"), "//"))
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "@commands":
			metadata.Name = value
			required["@commands"] = true
		case "@pkg_manager":
			metadata.PkgManager = strings.Split(value, ",")
			required["@pkg_manager"] = true
		case "@dependencies":
			if value != "null" {
				metadata.Dependencies = strings.Split(value, ",")
			}
		case "@author":
			metadata.Author = value
			required["@author"] = true
		case "@version":
			metadata.Version = value
			required["@version"] = true
		case "@description":
			metadata.Description = value
			required["@description"] = true
		}
	}

	// 驗證必填欄位
	for field, filled := range required {
		if !filled {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	route.Metadata = metadata
	return nil
}

// TODO: Implement route parsing and command execution 