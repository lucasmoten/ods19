package util

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"

	"go.uber.org/zap"
)

// getIP returns an IPv4 Address in string format suitable for Gatekeeper to reach us at
func GetIPRaw() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	if len(hostname) <= 0 {
		return "", fmt.Errorf("could not find our hostname: %s", hostname)
	}
	myIPs, err := net.LookupIP(hostname)
	if err != nil {
		log.Printf("could not get a set of IPs for our hostname: %v", err)
		log.Printf("try localhost")
		myIPs, err = net.LookupIP("localhost")
		if err != nil {
			log.Printf("failed to look up localhost: %v", err)
		}
	}
	if len(myIPs) <= 0 {
		return "", errors.New("could not find IPv4 address")
	}
	for a := range myIPs {
		if myIPs[a].To4() != nil {
			return myIPs[a].String(), nil
		}
	}
	return "", errors.New("could not find IPv4 address")
}

// GetIP wraps gm-fabric-go GetIP, optionally logs, and only returns the IP or empty string
func GetIP(logger *zap.Logger) string {
	ip, err := GetIPRaw()
	if err != nil {
		logger.Error("error getting ip", zap.Error(err))
	}
	return ip
}
