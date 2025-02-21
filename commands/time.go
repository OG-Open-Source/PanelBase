// @commands: time
// @pkg_manager: apk, apt, opkg, pacman, yum, zypper, dnf
// @dependencies: golang
// @author: PanelBase Team
// @version: 1.0.0
// @description: Show current time

package time

import (
	"fmt"
	"time"
)

func main() {
	now := time.Now()
	fmt.Println("當前時間：", now.Format("2006-01-02 15:04:05"))
}