/*
This package provides AES/CBC/PKCS5Padding encryption/decryption compatible
with the gov.ic.dodiis.commons.core.security.utils implementation.

This is based on:
* https://gitlab.363-283.io/dodiis-commons/commons-security/blob/develop/core-security/src/main/java/gov/ic/dodiis/commons/core/security/service/ResourceManagedTokenEncryption.java
* https://gitlab.363-283.io/dodiis-commons/commons-security/blob/develop/core-security/src/main/java/gov/ic/dodiis/commons/core/security/service/PBEBasedTokenEncryption.java
* https://gitlab.363-283.io/dodiis-commons/commons-security/blob/develop/core-security/src/main/java/gov/ic/dodiis/commons/core/security/service/util/TokenJarGenerator.java
* https://gitlab.363-283.io/dodiis-commons/commons-security/blob/develop/core-security/src/main/java/gov/ic/dodiis/commons/core/security/utils/GeneralPurposeEncryptor.java

Among other things, Bedrock's Puppet setup encrypts sensitive things like DB
passwords. The passwords are stored in a format like so:

	ENC{ZZZ}

Where ZZZ is:

	a hex encoding of
		a base64 encoding of
			an initialization vector (IV) followed by
			the original plaintext encrypted with AES/CBC/PKCS5Padding (using the preceding IV).

At present, most services (written in Java) locate a Jar file (token.jar) on the
classpath and extract from it an embedded file (sample.dat) that consists of the
following:

	The first byte is a 0 or 1, indicating an irrelevant (for our purposes) detail
	of how the key was derived.

	The remaining bytes are in the aforementioned base64+encrypted scheme.

The services then decrypt the sample.dat using the a common, shared key. This
yields a _new_ key that is then subsequently used to decrypt stuff like the
ENC{ZZZ} strings.
*/
package encryptor

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
)

var (
	ErrPaddingSize = errors.New("padding size error")
)

// TODO: randomly generate this
var iv = []byte{
	// This was pulled from Encryptor.java
	0xff, 0xef, 0xdf, 0xcf, 0x00, 0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80, 0x90, 0xa0, 0xb0,
}

const ivLength = 16

// Encrypt plaintext
func Encrypt(msg []byte, key []byte) ([]byte, error) {
	var block cipher.Block
	var err error

	// make sure we don't clobber the caller's slice.
	msg = dupBytes(msg)

	if block, err = aes.NewCipher(key); err != nil {
		return nil, fmt.Errorf("aes.NewCipher() error(%v)", err)
	}
	blockMode := cipher.NewCBCEncrypter(block, iv)
	padded := pkcs5Pad(msg, blockMode.BlockSize())
	if len(padded) < blockMode.BlockSize() || len(padded)%blockMode.BlockSize() != 0 {
		return nil, errors.New("length error")
	}
	blockMode.CryptBlocks( /*dst*/ padded /*src*/, padded)

	ivAndCipherText := make([]byte, ivLength+len(padded))
	copy(ivAndCipherText[:ivLength], iv)
	copy(ivAndCipherText[ivLength:], padded)

	return ivAndCipherText, nil
}

// Decrypt ciphertext
func Decrypt(ivAndCipherText []byte, key []byte) ([]byte, error) {
	var block cipher.Block
	var err error

	// make sure we don't clobber the caller's slice.
	cipherText := dupBytes(ivAndCipherText[ivLength:])
	iv := ivAndCipherText[:ivLength]

	if block, err = aes.NewCipher(key); err != nil {
		return nil, fmt.Errorf("aes.NewCipher() error(%v)", err)
	}
	blockMode := cipher.NewCBCDecrypter(block, iv)
	if len(cipherText) < blockMode.BlockSize() || len(cipherText)%blockMode.BlockSize() != 0 {
		return nil, errors.New("length error")
	}
	blockMode.CryptBlocks( /*dst*/ cipherText /*src*/, cipherText)
	msg, err := pkcs5Unpad(cipherText, blockMode.BlockSize())
	return msg, err
}

// Replace all occurences of ENC{....} with the decrypted contents thereof.
func ReplaceAll(str string, key []byte) (string, error) {
	strBytes := []byte(str)

	replaced := []byte{}

	for {
		startIndex := bytes.Index(strBytes, []byte("ENC{"))
		if startIndex == -1 {
			replaced = append(replaced, strBytes...)
			break
		} else {
			replaced = append(replaced, strBytes[:startIndex]...)
		}

		endIndex := bytes.Index(strBytes, []byte("}"))
		if endIndex == -1 {
			return "", errors.New("unmatched brace in ENC{}")
		}

		// Drop the "ENC{" from the front and the "}" at the end.
		hexBytes := strBytes[startIndex+4 : endIndex]

		hexDecoded, err := hex.DecodeString(string(hexBytes))
		if err != nil {
			return str, err
		}

		base64Decoded, err := base64.StdEncoding.DecodeString(string(hexDecoded))
		if err != nil {
			return str, err
		}

		dec, err := Decrypt(base64Decoded, key)
		if err != nil {
			return str, err
		}

		replaced = append(replaced, dec...)

		strBytes = strBytes[endIndex+1:]
	}

	return string(replaced), nil
}

func KeyFromTokenJar(path string, rootPassword string) ([]byte, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name != "sample.dat" {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		bytes, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, err
		}
		// drop the first flag byte; the IV and ciphertext follow.
		bytes = bytes[1:]

		base64Decoded, err := base64.StdEncoding.DecodeString(string(bytes))
		if err != nil {
			return nil, err
		}

		key, err := Decrypt(base64Decoded, []byte(rootPassword))
		if err != nil {
			return nil, err
		}

		return key, nil
	}

	return nil, errors.New("Could not find sample.dat in JAR file")
}

func EncryptParameter(val string, key []byte) (string, error) {
	ciphertext, err := Encrypt([]byte(val), key)
	if err != nil {
		return "", err
	}

	base64Encoded := base64.StdEncoding.EncodeToString(ciphertext)
	hexEncoded := hex.EncodeToString([]byte(base64Encoded))

	encoded := fmt.Sprintf("ENC{%s}", hexEncoded)
	return encoded, nil
}

// PKCS5 Padding/Unpadding
// -----------------------

func pkcs5Pad(src []byte, blockSize int) []byte {
	srcLen := len(src)
	padLen := blockSize - (srcLen % blockSize)
	padText := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(src, padText...)
}

func pkcs5Unpad(src []byte, blockSize int) ([]byte, error) {
	srcLen := len(src)
	paddingLen := int(src[srcLen-1])
	if paddingLen >= srcLen || paddingLen > blockSize {
		return nil, ErrPaddingSize
	}
	return src[:srcLen-paddingLen], nil
}

func dupBytes(slice []byte) []byte {
	c := make([]byte, len(slice))
	copy(c, slice)
	return c
}
