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
	Command string
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
		Log(EROR, "Failed to read routes config: %v", err)
		return err
	}

	var routes map[string]string
	if err := json.Unmarshal(data, &routes); err != nil {
		Log(EROR, "Failed to parse routes config: %v", err)
		return err
	}

	rm.routes = make(map[string]*Route)
	for code, command := range routes {
		rm.routes[code] = &Route{
			Command: command,
		}
	}

	Log(INFO, "Routes loaded successfully")
	return nil
}

func (rm *RouteManager) ExecuteCommand(route *Route, args map[string]string) (string, error) {
	tempDir, err := ioutil.TempDir("", "panelbase-cmd-")
	if err != nil {
		Log(EROR, "Failed to create temp directory: %v", err)
		return "", err
	}
	defer os.RemoveAll(tempDir)

	cmdPath := filepath.Join("internal/commands", route.Command)
	tempCmdPath := filepath.Join(tempDir, route.Command)
	if err := copyFile(cmdPath, tempCmdPath); err != nil {
		Log(EROR, "Failed to copy command file: %v", err)
		return "", err
	}

	if err := rm.parseCommandMetadata(route, tempCmdPath); err != nil {
		Log(EROR, "Failed to parse command metadata: %v", err)
		return "", err
	}

	cmdArgs := []string{}
	for key, value := range args {
		cmdArgs = append(cmdArgs, fmt.Sprintf("*#ARG_%s#*=%s", key, value))
	}

	cmd := exec.Command(tempCmdPath, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		Log(EROR, "Command execution failed: %v", err)
		return "", err
	}

	Log(INFO, "Command executed successfully: %s", route.Command)
	return string(output), nil
}

func (rm *RouteManager) GetRoute(path string) *Route {
	return rm.routes[path]
}

func copyFile(src, dst string) error {
	input, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, input, 0755)
}

func (rm *RouteManager) parseCommandMetadata(route *Route, cmdPath string) error {
	file, err := os.Open(cmdPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
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
		if _, exists := required[key]; exists {
			required[key] = true
		}
	}

	for field, filled := range required {
		if !filled {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	return nil
}

// TODO: Implement route parsing and command execution