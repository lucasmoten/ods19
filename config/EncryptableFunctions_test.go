package config

import (
	"testing"

	odrivecrypto "bitbucket.di2e.net/dime/object-drive-server/crypto"
)

var funcs = EncryptableFunctions{
	EncryptionStateBanner:  EncryptionBannerFalse,
	EncryptionStateHeader:  EncryptionHeaderFalse,
	DoCipherByReaderWriter: odrivecrypto.DoCipherByReaderWriter,
}

func TestEncryptionBanner(t *testing.T) {
	actual := funcs.EncryptionStateBanner()
	expected := EncryptionBannerFalse()
	if actual != expected {
		t.Errorf("EncryptionBanner())  %q, want %q", actual, expected)
	}
}

func TestNoopEncryptionBanner(t *testing.T) {
	var funcs = EncryptableFunctions{
		EncryptionStateBanner: EncryptionBannerTrue,
	}
	actual := funcs.EncryptionStateBanner()
	expected := EncryptionBannerTrue()
	if actual != expected {
		t.Errorf("Noop EncryptionBanner())  %q, want %q", actual, expected)
	}
}

func TestEncryptionWarning(t *testing.T) {
	var funcs = EncryptableFunctions{
		EncryptionStateHeader: EncryptionHeaderFalse,
	}

	actualKey, actualVal := funcs.EncryptionStateHeader()
	expectedKey, expectedVal := EncryptionHeaderFalse()
	if actualKey != expectedKey || actualVal != expectedVal {
		t.Errorf("EncryptionWarning())  %q, %q, want %q, %q", actualKey, actualVal, expectedKey, expectedVal)
	}
}

func TestNoopEncryptionWarning(t *testing.T) {
	var funcs = EncryptableFunctions{
		EncryptionStateHeader: EncryptionHeaderTrue,
	}
	actualKey, actualVal := funcs.EncryptionStateHeader()
	expectedKey, expectedVal := EncryptionHeaderTrue()
	if actualKey != expectedKey || actualVal != expectedVal {
		t.Errorf("Noop EncryptionWarning())  %q, %q, want %q, %q", actualKey, actualVal, expectedKey, expectedVal)
	}
}
