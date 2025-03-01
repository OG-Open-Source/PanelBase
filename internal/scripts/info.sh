#!/bin/bash
# @script: get_info
# @pkg_managers: apk, apt, opkg, pacman, yum, zypper, dnf
# @dependencies: bash
# @authors: PanelBase Team
# @version: 1.0.0
# @description: Get system information

[ -f ~/utilkit.sh ] && source /root/utilkit.sh || bash <(curl -sL raw.ogtt.tk/shell/get_utilkit.sh) && source /root/utilkit.sh
add_swap() {
	local new_swap=$1
	local swap_partitions=$(grep -E '^/dev/' /proc/swaps | awk '{print $1}')
	for partition in $swap_partitions; do
		swapoff "$partition"
		wipefs -a "$partition"
		mkswap -f "$partition"
	done
	swapoff /swapfile
	rm -f /swapfile
	fallocate -l ${new_swap}M /swapfile
	chmod 600 /swapfile
	mkswap /swapfile
	swapon /swapfile
	sed -i '/\/swapfile/d' /etc/fstab
	echo "/swapfile swap swap defaults 0 0" >> /etc/fstab
	if [ -f /etc/alpine-release ]; then
		echo "nohup swapon /swapfile" > /etc/local.d/swap.start
		chmod +x /etc/local.d/swap.start
		rc-update add local
	fi
}
case '*#ARG_1#*' in
"os") CHECK_OS ;;
"os_name") CHECK_OS -n ;;
"os_version") CHECK_OS -v ;;
"virt") CHECK_VIRT ;;
"cpu_cache") CPU_CACHE ;;
"cpu_freq") CPU_FREQ ;;
"cpu_model") CPU_MODEL ;;
"cpu_usage") CPU_USAGE ;;
"cpu_cores") nproc ;;
"disk_usage") DISK_USAGE ;;
"disk_usage_used") CONVERT_SIZE $(DISK_USAGE -u) ;;
"disk_usage_total") CONVERT_SIZE $(DISK_USAGE -t) ;;
"disk_usage_percentage") DISK_USAGE -p ;;
"dns_addr") DNS_ADDR ;;
"interface") INTERFACE ;;
"interface_rx_bytes") CONVERT_SIZE $(INTERFACE RX_BYTES) ;;
"interface_rx_packers") CONVERT_SIZE $(INTERFACE RX_PACKETS) ;;
"interface_rx_drop") CONVERT_SIZE $(INTERFACE RX_DROP) ;;
"interface_tx_bytes") CONVERT_SIZE $(INTERFACE TX_BYTES) ;;
"interface_tx_packers") CONVERT_SIZE $(INTERFACE TX_PACKETS) ;;
"interface_tx_drop") CONVERT_SIZE $(INTERFACE TX_DROP) ;;
"interface_info") INTERFACE -i ;;
"ip_addr") IP_ADDR ;;
"ip_addr_4") IP_ADDR -4 ;;
"ip_addr_6") IP_ADDR -6 ;;
"last_update") LAST_UPDATE ;;
"load_average") LOAD_AVERAGE ;;
"location") LOCATION ;;
"mac_addr") MAC_ADDR ;;
"mem_usage") MEM_USAGE ;;
"mem_usage_used") CONVERT_SIZE $(MEM_USAGE -u) ;;
"mem_usage_total") CONVERT_SIZE $(MEM_USAGE -t) ;;
"mem_usage_percentage") MEM_USAGE -p ;;
"net_provider") NET_PROVIDER ;;
"pkg_count") PKG_COUNT ;;
"public_ip") PUBLIC_IP ;;
"shell_ver") SHELL_VER ;;
"swap_usage") SWAP_USAGE ;;
"swap_usage_used") CONVERT_SIZE $(SWAP_USAGE -u) ;;
"swap_usage_total") CONVERT_SIZE $(SWAP_USAGE -t) ;;
"swap_usage_percentage") SWAP_USAGE -p ;;
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
"get_tcp_congestion") sysctl -n net.ipv4.tcp_congestion_control ;;
"get_tcp_congestion_list") sysctl -n net.ipv4.tcp_available_congestion_control ;;
"get_tcp_qdisc") sysctl -n net.core.default_qdisc ;;
"get_tcp_congestion_and_qdisc") echo $(sysctl -n net.ipv4.tcp_congestion_control) $(sysctl -n net.core.default_qdisc) ;;
"time") echo $(TIMEZONE -i) $(date +"%Y-%m-%d %H:%M:%S") ;;
"swap_setup") INPUT 'Please enter the swap size (MiB):' swap  && add_swap $swap ;;
esac
