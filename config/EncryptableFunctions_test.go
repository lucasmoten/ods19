package config

import (
	"testing"

	odrivecrypto "bitbucket.di2e.net/dime/object-drive-server/crypto"
)

var funcs = EncryptableFunctions{
	EncryptionBanner:       EncryptionBannerF,
	EncryptionWarning:      EncryptionWarningF,
	DoCipherByReaderWriter: odrivecrypto.DoCipherByReaderWriter,
}

func TestEncryptionBanner(t *testing.T) {
	actual := funcs.EncryptionBanner()
	expected := EncryptionBannerF()
	if actual != expected {
		t.Errorf("EncryptionBanner())  %q, want %q", actual, expected)
	}
}

func TestNoopEncryptionBanner(t *testing.T) {
	var funcs = EncryptableFunctions{
		EncryptionBanner: NoopEncryptionBannerF,
	}
	actual := funcs.EncryptionBanner()
	expected := NoopEncryptionBannerF()
	if actual != expected {
		t.Errorf("Noop EncryptionBanner())  %q, want %q", actual, expected)
	}
}

func TestEncryptionWarning(t *testing.T) {
	var funcs = EncryptableFunctions{
		EncryptionWarning: EncryptionWarningF,
	}

	actualKey, actualVal := funcs.EncryptionWarning()
	expectedKey, expectedVal := EncryptionWarningF()
	if actualKey != expectedKey || actualVal != expectedVal {
		t.Errorf("EncryptionWarning())  %q, %q, want %q, %q", actualKey, actualVal, expectedKey, expectedVal)
	}
}

func TestNoopEncryptionWarning(t *testing.T) {
	var funcs = EncryptableFunctions{
		EncryptionWarning: NoopEncryptionWarningF,
	}
	actualKey, actualVal := funcs.EncryptionWarning()
	expectedKey, expectedVal := NoopEncryptionWarningF()
	if actualKey != expectedKey || actualVal != expectedVal {
		t.Errorf("Noop EncryptionWarning())  %q, %q, want %q, %q", actualKey, actualVal, expectedKey, expectedVal)
	}
}
