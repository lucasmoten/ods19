package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"math/big"
)

// ByteRange for handling video
type ByteRange struct {
	Start int64
	Stop  int64
}

//NewByteRange is the implicit default byte range that we have always used
func NewByteRange() *ByteRange {
	br := &ByteRange{}
	br.Stop = -1
	return br
}

// CipherStreamReader takes statistics as it writes
type CipherStreamReader struct {
	S       cipher.Stream
	R       io.Reader
	H       hash.Hash
	Size    int64
	Written int64
}

// NewCipherStreamReader Create a new ciphered stream with hashing
func NewCipherStreamReader(w cipher.Stream, r io.Reader) *CipherStreamReader {
	return &CipherStreamReader{
		S:       w,
		R:       r,
		H:       sha256.New(),
		Size:    int64(0),
		Written: int64(0),
	}
}

// Read takes statistics as it writes
func (r *CipherStreamReader) Read(dst []byte) (n int, err error) {
	n, err = r.R.Read(dst)
	if err != nil {
		if err == io.EOF {
			//nothing is wrong
		} else {
			log.Printf("error while reading from cipher stream:%v at size %d", err, r.Size)
		}
	}
	r.H.Write(dst[:n])
	r.S.XORKeyStream(dst[:n], dst[:n])
	r.Size += int64(n)
	r.Written += int64(n)
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

func CreateRandomName() string {
	//Sha256 keys are 256 bits
	key := make([]byte, 32)
	rand.Read(key)
	return hex.EncodeToString(key)
}

// ApplyPassPhrsae takes the passphrase provided and performs a bitwise XOR on
// each element of the current contents of the passed in array
func ApplyPassphrase(passphrase string, fileKey []byte) {
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

// CreateKey creates a byte array of length 32 initialized with random data
func CreateKey() (key []byte) {
	//256 bit keys
	key = make([]byte, 32)
	rand.Read(key)
	return
}

// CreateIV creates an byte array representing the initialization vector of the
// same size as the AES Block Size.
func CreateIV() (iv []byte) {
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

// DoCipherByReaderWriter initializes a new AES cipher with the provided key
// and initialization vector reading from io.Reader, applying the cipher
// and writing to the io.Writer.
//XXX Need a proper read-write pipe that will xor with the key as it writes,
// need to facilitate efficient encrypted append.
func DoCipherByReaderWriter(
	inFile io.Reader,
	outFile io.Writer,
	key []byte,
	iv []byte,
	description string,
	byteRange *ByteRange,
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

	length, err = RangeCopy(outFile, reader, byteRange)
	return reader.H.Sum(nil), reader.Size, err
}

// RangeCopy use begin for first byte location, and end is beyond the one we copy,
// to be more like other APIs
func RangeCopy(dst io.Writer, src io.Reader, byteRange *ByteRange) (int64, error) {

	if byteRange == nil {
		return io.Copy(dst, src)
	}

	var err error
	if byteRange.Start > int64(0) {
		_, err = io.CopyN(ioutil.Discard, src, byteRange.Start)
		if err != nil {
			return 0, err
		}
	}
	if byteRange.Stop == -1 {
		return io.Copy(dst, src)
	}
	rangeDiff := byteRange.Stop - byteRange.Start + 1
	return io.CopyN(dst, src, rangeDiff)
}