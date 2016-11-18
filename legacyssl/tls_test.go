package legacyssl

import "testing"

func TestSetOpenSSLDialOptions(t *testing.T) {
	opts := OpenSSLDialOptions{}

	opts.SetInsecureSkipHostVerification()
	if opts.Flags != 1 {
		t.Error("Expected Flag 1 to be set, got: ", opts.Flags)
	}

	opts.SetDisableSNI()
	if opts.Flags != 3 {
		t.Error("Expected Flag 1 and Flag 2 to be set, bitmask is: ", opts.Flags)
	}

}
