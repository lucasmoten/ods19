package crypto

import (
	"crypto/sha256"
	"hash"
	"io"

	"go.uber.org/zap"
)

// plainStreamReader takes statistics as it writes
type plainStreamReader struct {
	S       io.Writer
	R       io.Reader
	H       hash.Hash
	Size    int64
	Written int64
	Logger  *zap.Logger
}

// Read takes statistics as it writes
func (r *plainStreamReader) Read(dst []byte) (n int, err error) {
	n, err = r.R.Read(dst)
	if err != nil {
		if err == io.EOF {
			//nothing is wrong
		} else {
			return n, err
		}
	}
	r.H.Write(dst[:n])
	r.Size += int64(n)
	r.Written += int64(n)
	return
}

// newPlainStreamReader Create a new unciphered stream with hashing
func newPlainStreamReader(logger *zap.Logger, w io.Writer, r io.Reader) *plainStreamReader {
	return &plainStreamReader{
		S:       w,
		R:       r,
		H:       sha256.New(),
		Size:    int64(0),
		Written: int64(0),
		Logger:  logger,
	}
}

// DoNocipherByReaderWriter reads from io.Reader and writes to the io.Writer and takes stats as it is doing it.
func DoNocipherByReaderWriter(
	logger *zap.Logger,
	inFile io.Reader,
	outFile io.Writer,
	key []byte,
	iv []byte,
	description string,
	byteRange *ByteRange,
) (checksum []byte, length int64, err error) {
	reader := newPlainStreamReader(logger, outFile, inFile)

	length, err = rangeCopy(outFile, reader, byteRange)
	return reader.H.Sum(nil), reader.Size, err
}
