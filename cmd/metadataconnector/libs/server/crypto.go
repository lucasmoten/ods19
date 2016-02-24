package server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"io"
	"log"
	"math/big"
)

// CipherStreamReader takes statistics as it writes
type CipherStreamReader struct {
	S    cipher.Stream
	R    io.Reader
	H    hash.Hash
	Size int64
}

// NewCipherStreamReader Create a new ciphered stream with hashing
func NewCipherStreamReader(w cipher.Stream, r io.Reader) *CipherStreamReader {
	return &CipherStreamReader{
		S:    w,
		R:    r,
		H:    sha256.New(),
		Size: int64(0),
	}
}

// Read takes statistics as it writes
func (r *CipherStreamReader) Read(dst []byte) (n int, err error) {
	n, err = r.R.Read(dst)
	r.H.Write(dst[:n])
	r.S.XORKeyStream(dst[:n], dst[:n])
	r.Size += int64(n)
	////XXX not good for performance, but we are getting cut-offs, and this
	////is insightful to uncomment
	//log.Printf("transferred:%d to %d", int64(n), r.Size)
	return
}

//RSAComponents is effectively a parsed and unlocked pkcs12 store
//It is the actual numbers required to do RSA computations
type RSAComponents struct {
	N *big.Int
	D *big.Int
	E *big.Int
}

func parseRSAComponents(nStr, dStr, eStr string) (*RSAComponents, error) {
	nBytes, err := base64.StdEncoding.DecodeString(nStr)
	if err != nil {
		log.Printf("Unable to parse RSA component N")
		return nil, err
	}
	var n big.Int
	n.SetBytes(nBytes)

	dBytes, err := base64.StdEncoding.DecodeString(dStr)
	if err != nil {
		log.Printf("Unable to parse RSA component D")
		return nil, err
	}
	var d big.Int
	d.SetBytes(dBytes)

	eBytes, err := base64.StdEncoding.DecodeString(eStr)
	if err != nil {
		log.Printf("Unable to parse RSA component E")
		return nil, err
	}
	var e big.Int
	e.SetBytes(eBytes)

	return &RSAComponents{
		N: &n,
		D: &d,
		E: &e,
	}, nil
}

func createRSAComponents(randReader io.Reader) (*RSAComponents, error) {
	//TODO: keysize must be a parameter
	rsaPair, err := rsa.GenerateKey(randReader, 2048)
	if err != nil {
		log.Printf("Unable to generate RSA keypair")
		return nil, err
	}
	return &RSAComponents{
		N: rsaPair.N,
		D: rsaPair.D,
		E: big.NewInt(int64(rsaPair.E)),
	}, nil
}

//Generate unique opaque names for uploaded files
//This would be straight base64 encoding, except the characters need
//to be valid filenames
func obfuscateHash(key string) string {
	hashBytes := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hashBytes[:])
}

func createRandomName() string {
	//Sha256 keys are 256 bits
	key := make([]byte, 32)
	rand.Read(key)
	return hex.EncodeToString(key)
}

func applyPassphrase(passphrase string, fileKey []byte) {
	hashBytes := sha256.Sum256([]byte(passphrase))
	fklen := len(fileKey)
	hlen := len(hashBytes)
	if fklen > hlen {
		//If we conveniently use this to encrypt long data, it's effectively
		//ECB mode without some changes.  Don't use for more than keys
		log.Fatal("Do not applyPassphrase to anything that is longer than a sha256 hash!")
	}
	for i := 0; i < fklen; i++ {
		fileKey[i] = hashBytes[i] ^ fileKey[i]
	}
	return
}

func createKey() (key []byte) {
	//256 bit keys
	key = make([]byte, 32)
	rand.Read(key)
	return
}

func createIV() (iv []byte) {
	//XXX I have read advice that with CTR blocks, the last four bytes
	//of an iv should be zero, because the last four bytes are
	//actually a counter for - seeking in the stream?
	//That may allow appending to files - tbd
	//Also note that we have fewer issues with iv sizes being large
	//enough due to using this to encrypt random keys.
	iv = make([]byte, aes.BlockSize)
	rand.Read(iv)
	iv[aes.BlockSize-1] = 0
	iv[aes.BlockSize-2] = 0
	iv[aes.BlockSize-3] = 0
	iv[aes.BlockSize-4] = 0
	return
}

//XXX Need a proper read-write pipe that will xor with the key as it writes,
// need to facilitate efficient encrypted append.
func doCipherByReaderWriter(
	inFile io.Reader,
	outFile io.Writer,
	key []byte,
	iv []byte,
	description string,
) (checksum []byte, length int64, err error) {
	writeCipher, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("unable to use cipher %s: %v", description, err)
		return nil, 0, err
	}
	writeCipherStream := cipher.NewCTR(writeCipher, iv[:])
	if err != nil {
		log.Printf("unable to use block mode (%s):%v", description, err)
		return nil, 0, err
	}

	reader := NewCipherStreamReader(writeCipherStream, inFile)
	_, err = io.Copy(outFile, reader)
	if err != nil {
		log.Printf("unable to copy out to file (%s):%v", description, err)
	}
	return reader.H.Sum(nil), reader.Size, err
}

func doReaderWriter(inFile io.Reader, outFile io.Writer) error {
	_, err := io.Copy(outFile, inFile)
	return err
}
