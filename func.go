package limiter

import "net"

// Extracts the IP from a string containing both port and IP
func extractIP(remote string) string {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return ""
	}
	return host
}
