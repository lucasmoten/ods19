package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"

	"go.uber.org/zap"
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

// cipherStreamReader takes statistics as it writes
type cipherStreamReader struct {
	S       cipher.Stream
	R       io.Reader
	H       hash.Hash
	Size    int64
	Written int64
	Logger  *zap.Logger
}

// newCipherStreamReader Create a new ciphered stream with hashing
func newCipherStreamReader(logger *zap.Logger, w cipher.Stream, r io.Reader) *cipherStreamReader {
	return &cipherStreamReader{
		S:       w,
		R:       r,
		H:       sha256.New(),
		Size:    int64(0),
		Written: int64(0),
		Logger:  logger,
	}
}

// Read takes statistics as it writes
func (r *cipherStreamReader) Read(dst []byte) (n int, err error) {
	n, err = r.R.Read(dst)
	if err != nil {
		if err == io.EOF {
			//nothing is wrong
		} else {
			return n, err
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

// CreateRandomName gives each file a random name
func CreateRandomName() string {
	key := make([]byte, 32)
	rand.Read(key)
	return hex.EncodeToString(key)
}

// DoMAC generates a repeatable hash value from the concatenation of key components of the permission, to be used as a message authentication code
func DoMAC(passphrase string, permissionIV []byte, grantee string, c, r, u, d, s bool, encryptedKey []byte) []byte {
	//TODO: use value types
	result := make([]byte, 32)
	//!!! the last item is of a fixed length because we are of the form H(secret||data),
	// so we don't want to have issues with extension attacks on the hash
	str := fmt.Sprintf("%s:%s:%t,%t,%t,%t,%t:%s", passphrase, grantee, c, r, u, d, s, hex.EncodeToString(encryptedKey))
	hashBytes := sha256.Sum256([]byte(str))
	for i := 0; i < 32; i++ {
		result[i] = hashBytes[i]
	}
	return result
}

// ApplyPassphrase takes the passphrase provided and performs a bitwise XOR on
// each element of the current contents of the passed in array
func ApplyPassphrase(passphrase string, permissionIV, fileKey []byte) []byte {
	result := make([]byte, 32)
	//!!! the last item is of a fixed length because we are of the form H(secret||data),
	// so we don't want to have issues with extension attacks on the hash
	str := fmt.Sprintf("%s:%s", passphrase, hex.EncodeToString(permissionIV))
	hashBytes := sha256.Sum256([]byte(str))
	fklen := len(fileKey)
	hlen := len(hashBytes)
	if fklen > hlen {
		//If we conveniently use this to encrypt long data, it's effectively
		//ECB mode without some changes.  Don't use for more than keys
		log.Fatal("Do not applyPassphrase to anything that is longer than a sha256 hash!")
	}
	for i := 0; i < fklen; i++ {
		result[i] = hashBytes[i] ^ fileKey[i]
	}
	return result
}

// CreatePermissionIV creates a byte array of length 32 initialized with random data
func CreatePermissionIV() (key []byte) {
	//256 bit keys
	key = make([]byte, 32)
	rand.Read(key)
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
	logger *zap.Logger,
	inFile io.Reader,
	outFile io.Writer,
	key []byte,
	iv []byte,
	description string,
	byteRange *ByteRange,
) (checksum []byte, length int64, err error) {
	writeCipher, err := aes.NewCipher(key)
	if err != nil {
		logger.Error("unable to use cipher", zap.String("description", description), zap.Error(err))
		return nil, 0, err
	}
	writeCipherStream := cipher.NewCTR(writeCipher, iv[:])
	if err != nil {
		logger.Error("unable to use block mode", zap.String("description", description), zap.Error(err))
		return nil, 0, err
	}

	reader := newCipherStreamReader(logger, writeCipherStream, inFile)

	length, err = rangeCopy(outFile, reader, byteRange)
	return reader.H.Sum(nil), reader.Size, err
}

// rangeCopy use begin for first byte location, and end is beyond the one we copy,
// to be more like other APIs
func rangeCopy(dst io.Writer, src io.Reader, byteRange *ByteRange) (int64, error) {

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
