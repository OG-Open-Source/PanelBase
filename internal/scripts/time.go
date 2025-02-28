// @script: time
// @pkg_managers: apk, apt, opkg, pacman, yum, zypper, dnf
// @dependencies: golang
// @authors: PanelBase Team
// @version: 1.0.0
// @description: Returns the current server time

package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Print(time.Now().Format("2006-01-02T15:04:05Z"))
}