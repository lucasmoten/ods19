package util

import (
	"crypto/rand"
	"fmt"
	"io"
)

// NewGUID generates string representations of GUIDs.
func NewGUID() (string, error) {
	guid, err := NewGUIDBytes()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x%x%x%x%x", guid[0:4], guid[4:6], guid[6:8], guid[8:10], guid[10:]), nil
}

// NewGUIDBytes generates byte representations of GUIDs.
func NewGUIDBytes() ([]byte, error) {
	guid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, guid)
	if n != len(guid) || err != nil {
		return guid, err
	}
	guid[8] = guid[8]&^0xc0 | 0x80
	guid[6] = guid[6]&^0xf0 | 0x40

	return guid, nil
}
