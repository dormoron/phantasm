package host

import (
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	// ErrInvalidHost 无效的主机名错误
	ErrInvalidHost = errors.New("无效的主机名")

	// ErrInvalidPort 无效的端口号错误
	ErrInvalidPort = errors.New("无效的端口号")
)

// ExtractHostPort 从地址中提取主机名和端口号
func ExtractHostPort(addr string) (host string, port int, err error) {
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	if h == "" {
		return "", 0, ErrInvalidHost
	}

	var portNum int
	if p != "" {
		portNum, err = strconv.Atoi(p)
		if err != nil || portNum < 0 || portNum > 65535 {
			return "", 0, ErrInvalidPort
		}
	}

	return h, portNum, nil
}

// IsLocalHost 判断是否为本地主机地址
func IsLocalHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// GetLocalIP 获取本地IP地址
func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", errors.New("无法获取本地IP地址")
}

// GetHostname 获取主机名
func GetHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	// 清理主机名中的特殊字符
	hostname = strings.TrimSpace(hostname)

	return hostname, nil
}

// PortAvailable 检查端口是否可用
func PortAvailable(port int) bool {
	if port <= 0 || port > 65535 {
		return false
	}

	conn, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return false
	}

	conn.Close()
	return true
}

// FindAvailablePort 寻找可用端口
func FindAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", ":0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

// BuildAddress 根据主机名和端口号构建地址
func BuildAddress(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// MustGetLocalIP 必须获取本地IP，如果获取失败则返回备用IP
func MustGetLocalIP(fallback string) string {
	ip, err := GetLocalIP()
	if err != nil {
		return fallback
	}
	return ip
}

// IsIPv6 判断是否为IPv6地址
func IsIPv6(addr string) bool {
	ip := net.ParseIP(addr)
	return ip != nil && ip.To4() == nil
}

// IsIPv4 判断是否为IPv4地址
func IsIPv4(addr string) bool {
	ip := net.ParseIP(addr)
	return ip != nil && ip.To4() != nil
}
