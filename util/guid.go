package util

import (
	"crypto/rand"
	"fmt"
	"io"
)

// NewGUID can generate string representations of GUIDs for testing
func NewGUID() (string, error) {
	guid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, guid)
	if n != len(guid) || err != nil {
		return "", err
	}
	guid[8] = guid[8]&^0xc0 | 0x80
	guid[6] = guid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x%x%x%x%x", guid[0:4], guid[4:6], guid[6:8], guid[8:10], guid[10:]), nil
}
