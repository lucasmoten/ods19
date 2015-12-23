package libs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log"
	"math/big"
	"strings"
)

// CountingStreamReader takes statistics as it writes
type CountingStreamReader struct {
	S cipher.Stream
	R io.Reader
}

// Read takes statistics as it writes
func (r CountingStreamReader) Read(dst []byte) (n int, err error) {
	n, err = r.R.Read(dst)
	r.S.XORKeyStream(dst[:n], dst[:n])
	return
}

// CountingStreamWriter keeps statistics as it writes
type CountingStreamWriter struct {
	S     cipher.Stream
	W     io.Writer
	Error error
}

func (w CountingStreamWriter) Write(src []byte) (n int, err error) {
	c := make([]byte, len(src))
	w.S.XORKeyStream(c, src)
	n, err = w.W.Write(c)
	if n != len(src) {
		if err == nil {
			err = io.ErrShortWrite
		}
	}
	return
}

// Close closes underlying stream
func (w CountingStreamWriter) Close() error {
	if c, ok := w.W.(io.Closer); ok {
		return c.Close()
	}
	return nil
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
		keyString := base64.StdEncoding.EncodeToString(hashBytes[:])
		return strings.Replace(strings.Replace(keyString, "/", "~", -1), "=", "Z", -1)
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

func (h Uploader) retrieveMetaData(fileName string, dn string) (key []byte, iv []byte, cls []byte, err error) {
	keyFileName := fileName + "_" + obfuscateHash(dn) + ".key"
	ivFileName := fileName + ".iv"
	classFileName := fileName + ".class"

	classFile, closer, err := h.Backend.GetReadHandle(classFileName)
	if err != nil {
		return key, iv, cls, err
	}
	defer closer.Close()
	cls = make([]byte, 80)
	classFile.Read(cls)

	keyFile, closer, err := h.Backend.GetReadHandle(keyFileName)
	if err != nil {
		return key, iv, cls, err
	}
	defer closer.Close()
	key = make([]byte, h.KeyBytes)
	keyFile.Read(key)

	ivFile, closer, err := h.Backend.GetReadHandle(ivFileName)
	if err != nil {
		return key, iv, cls, err
	}
	defer closer.Close()
	iv = make([]byte, aes.BlockSize)
	ivFile.Read(iv)

	applyPassphrase([]byte(masterKey), key)
	return key, iv, cls, nil
}

func doCipherByReaderWriter(inFile io.Reader, outFile io.Writer, key []byte, iv []byte) error {
	writeCipher, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("unable to use cipher: %v", err)
		return err
	}
	writeCipherStream := cipher.NewCTR(writeCipher, iv[:])
	if err != nil {
		log.Printf("unable to use block mode:%v", err)
		return err
	}

	reader := &CountingStreamReader{S: writeCipherStream, R: inFile}
	_, err = io.Copy(outFile, reader)
	if err != nil {
		log.Printf("unable to copy out to file:%v", err)
	}
	return err
}

func doReaderWriter(inFile io.Reader, outFile io.Writer) error {
	_, err := io.Copy(outFile, inFile)
	return err
}
