package util

import (
	"github.com/deciphernow/gm-fabric-go/gk"
	"github.com/uber-go/zap"
)

// GetIP wraps gm-fabric-go GetIP, optionally logs, and only returns the IP or empty string
func GetIP(logger zap.Logger) string {
	ip, err := gk.GetIP()
	if err != nil {
		if logger != nil {
			logger.Error("error getting ip", zap.Error(err))
		}
		return ""
	}
	return ip
}