package libs

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
	S cipher.Stream
	R io.Reader
	H hash.Hash
}

// NewCipherStreamReader Create a new ciphered stream with hashing
func NewCipherStreamReader(w cipher.Stream, r io.Reader) *CipherStreamReader {
	return &CipherStreamReader{
		S: w,
		R: r,
		H: sha256.New(),
	}
}

// Read takes statistics as it writes
func (r CipherStreamReader) Read(dst []byte) (n int, err error) {
	n, err = r.R.Read(dst)
	r.S.XORKeyStream(dst[:n], dst[:n])
	r.H.Write(dst[:n])
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
	if hideFileNames {
		hashBytes := sha256.Sum256([]byte(key))
		return hex.EncodeToString(hashBytes[:])
	}
	return key
}

func (h Uploader) createKeyIVPair() (key []byte, iv []byte) {
	key = make([]byte, h.KeyBytes)
	rand.Read(key)
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

func doCipherByReaderWriter(
	inFile io.Reader,
	outFile io.Writer,
	key []byte,
	iv []byte,
) (checksum []byte, err error) {
	writeCipher, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("unable to use cipher: %v", err)
		return nil, err
	}
	writeCipherStream := cipher.NewCTR(writeCipher, iv[:])
	if err != nil {
		log.Printf("unable to use block mode:%v", err)
		return nil, err
	}

	reader := NewCipherStreamReader(writeCipherStream, inFile)
	_, err = io.Copy(outFile, reader)
	if err != nil {
		log.Printf("unable to copy out to file:%v", err)
	}
	return reader.H.Sum(nil), err
}

func doReaderWriter(inFile io.Reader, outFile io.Writer) error {
	_, err := io.Copy(outFile, inFile)
	return err
}
