#!/bin/bash
# @script: get_info
# @pkg_managers: apk, apt, opkg, pacman, yum, zypper, dnf
# @dependencies: bash
# @authors: PanelBase Team
# @version: 1.0.0
# @description: Get system information

[ -f ~/utilkit.sh ] && source /root/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source /root/utilkit.sh
case '*#ARG_1#*' in
"os") CHECK_OS ;;
"os_name") CHECK_OS -n ;;
"os_version") CHECK_OS -v ;;
"virt") CHECK_VIRT ;;
"cpu_cache") CPU_CACHE ;;
"cpu_freq") CPU_FREQ ;;
"cpu_model") CPU_MODEL ;;
"cpu_usage") CPU_USAGE ;;
"disk_usage") DISK_USAGE ;;
"dns_addr") DNS_ADDR ;;
"interface") INTERFACE ;;
"interface_rx_bytes") INTERFACE RX_BYTES ;;
"interface_rx_packers") INTERFACE RX_PACKETS ;;
"interface_rx_drop") INTERFACE RX_DROP ;;
"interface_tx_bytes") INTERFACE TX_BYTES ;;
"interface_tx_packers") INTERFACE TX_PACKETS ;;
"interface_tx_drop") INTERFACE TX_DROP ;;
"interface_info") INTERFACE -i ;;
"ip_addr") IP_ADDR ;;
"ip_addr_4") IP_ADDR -4 ;;
"ip_addr_6") IP_ADDR -6 ;;
"last_update") LAST_UPDATE ;;
"load_average") LOAD_AVERAGE ;;
"location") LOCATION ;;
"mac_addr") MAC_ADDR ;;
"mem_usage") MEM_USAGE ;;
"net_provider") NET_PROVIDER ;;
"pkg_count") PKG_COUNT ;;
"public_ip") PUBLIC_IP ;;
"shell_ver") SHELL_VER ;;
"swap_usage") SWAP_USAGE ;;
"sys_clean") SYS_CLEAN ;;
"sys_info") SYS_INFO ;;
"sys_optimize") SYS_OPTIMIZE ;;
"sys_reboot") SYS_REBOOT ;;
"sys_update") SYS_UPDATE ;;
"sys_upgrade") SYS_UPGRADE ;;
"timezone_internal") TIMEZONE -i ;;
"timezone_external") TIMEZONE -e ;;
"hostname") hostname ;;
"kernel") uname -r ;;
"uptime") uptime -p | sed 's/up //' ;;
esac
