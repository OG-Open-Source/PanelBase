# @commands: date
# @pkg_manager: apk, apt, opkg, pacman, yum, zypper, dnf
# @dependencies: python3
# @author: PanelBase Team
# @version: 1.0.0
# @description: Show current date

import datetime

today = datetime.date.today()

formatted_date = today.strftime("%Y-%m-%d")

print("Current Date: ", formatted_date)