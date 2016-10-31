package crypto_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/crypto"
)

func TestBasicCipher(t *testing.T) {
	logger := zap.New(zap.NewJSONEncoder())

	data := []byte(`
    0123456789
    0123456789
    0123456789
    0123456789
    0123456789
    0123456789
    0123456789
    0123456789
    0123456789
    0123456789
    0123456789
    `)
	byteRange := &crypto.ByteRange{
		Start: 0,
		Stop:  int64(len(data) - 1),
	}
	key := crypto.CreateKey()
	iv := crypto.CreateIV()

	plaintextName := "crypto_test.plain.tmp"
	ciphertextName := "crypto_test.cipher.tmp"
	defer os.Remove(ciphertextName)
	defer os.Remove(plaintextName)

	//Make the file exist in a (closed file)
	fPlain, err := os.Create(plaintextName)
	if err != nil {
		t.Errorf("Failed to open plaintext for write:%v", err)
	}
	fPlain.Write(data)
	fPlain.Close()

	//Open the file for read
	fPlain, err = os.Open(plaintextName)
	if err != nil {
		t.Errorf("Failed to open plaintext for read:%v", err)
	}
	defer fPlain.Close()

	//Prepare to write ciphertext
	fCipher, err := os.Create(ciphertextName)
	if err != nil {
		t.Errorf("Failed to open ciphertext for write:%v", err)
	}
	defer fCipher.Close()

	//Run the plaintext to get ciphertext
	checksum, length, err := crypto.DoCipherByReaderWriter(logger, fPlain, fCipher, key, iv, "write", byteRange)
	if err != nil {
		t.Errorf("Failed to compute full ciphertext:%v", err)
	}
	//This is the checksum of the *plaintext*, making it independent of key
	hexChecksum := hex.EncodeToString(checksum)
	hexChecksumExpected := "c998b62e0950a5529d0493f469eac818be596532a6e8b3fcfa8aa1c55c4efe19"
	if hexChecksum != hexChecksumExpected {
		t.Errorf("Checksum came out wrong: %s", hexChecksum)
	}
	if length != 170 {
		t.Errorf("%v", fmt.Errorf("wrong length %d", length))
	}
	fCipher.Close()

	//This is the easy case with a default range
	BasicCipherRaw(t, data, ciphertextName, byteRange, key, iv)

	//The first block is dropped and second truncated
	byteRange.Start = 35
	BasicCipherRaw(t, data, ciphertextName, byteRange, key, iv)

	//The last block is truncated
	byteRange.Stop = 150
	BasicCipherRaw(t, data, ciphertextName, byteRange, key, iv)

	//The a truncated block followed by dropped blocks
	byteRange.Stop = 120
	BasicCipherRaw(t, data, ciphertextName, byteRange, key, iv)

	//The a truncated block followed by dropped blocks
	byteRange.Start = 65
	BasicCipherRaw(t, data, ciphertextName, byteRange, key, iv)

}

func BasicCipherRaw(t *testing.T, data []byte, ciphertextName string, byteRange *crypto.ByteRange, key []byte, iv []byte) {
	var err error
	logger := zap.New(zap.NewJSONEncoder())

	//Make a temp file that we can close and re-open later.
	replaintextName := "crypto_test.replaintext.tmp"
	defer os.Remove(replaintextName)

	fCipher, err := os.Open(ciphertextName)
	if err != nil {
		t.Errorf("unable to reopen ciphertext:%v", err)
	}
	defer fCipher.Close()

	//Prepare to write recovered plaintext
	fReplain, err := os.Create(replaintextName)
	if err != nil {
		t.Errorf("Failed to open recovered plaintext for write:%v", err)
	}
	defer fReplain.Close()

	//Generate plaintext again
	_, _, err = crypto.DoCipherByReaderWriter(logger, fCipher, fReplain, key, iv, "reread", byteRange)
	fReplain.Close()

	//Read replain into a variable and compare it with expected result.
	fReplain, err = os.Open(replaintextName)
	if err != nil {
		t.Errorf("Failed to reopen recovered plaintext for read: %v", err)
	}
	defer fReplain.Close()

	reData, err := ioutil.ReadAll(fReplain)
	if err != nil {
		t.Errorf("Failed to re-read replaintext into byte array:%v", err)
	}

	if bytes.Compare(reData, data[byteRange.Start:byteRange.Stop+1]) != 0 {
		t.Errorf("Recovered data not the same for range:%d-%d", byteRange.Start, byteRange.Stop)
	}
}
