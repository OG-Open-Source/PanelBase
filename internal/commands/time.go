// @commands: time
// @pkg_manager: apk, apt, opkg, pacman, yum, zypper, dnf
// @dependencies: golang
// @author: PanelBase Team
// @version: 1.0.0
// @description: Returns the current server time

package commands

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"))
}