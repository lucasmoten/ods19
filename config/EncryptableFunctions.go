package config

import (
	"io"

	odrivecrypto "bitbucket.di2e.net/dime/object-drive-server/crypto"

	"go.uber.org/zap"
)

//Not Pure
type DoCipherByReaderWriter func(
	logger *zap.Logger,
	inFile io.Reader,
	outFile io.Writer,
	key []byte,
	iv []byte,
	description string,
	byteRange *odrivecrypto.ByteRange,
) (checksum []byte, length int64, err error)

//Pure
type EncryptionStateHeader func() (string, string)

//Pure
type EncryptionStateBanner func() string

type EncryptableFunctions struct {
	EncryptionStateBanner  EncryptionStateBanner
	EncryptionStateHeader  EncryptionStateHeader
	DoCipherByReaderWriter DoCipherByReaderWriter
}

func EncryptionHeaderFalse() (key, value string) {
	return "Content-Encrypted-At-Rest", "FALSE. The service is running without encrypting data at rest. Files are encrypted in transit only."
}

func EncryptionHeaderTrue() (key, value string) {
	return "Content-Encrypted-At-Rest", "TRUE"
}

func EncryptionBannerTrue() string {
	return "encryption of data at rest enabled"
}

func EncryptionBannerFalse() string {
	return " \n" +
		"============================================================ \n" +
		"============================================================ \n" +
		"============================================================ \n" +
		"                    W  A  R  N  I  N  G \n \n" +
		"This service is running without encryption at rest enabled. \n \n" +
		"This means that data that is uploaded to the service will be \n" +
		"stored in the local cache in an unencrypted, plain text form. \n" +
		" \n" +
		"Any data stored in S3 buckets will also be in plain text and \n" +
		"anyone with read access to that bucket directly, or via IAM \n" +
		"roles, will be able to see the raw content without being \n" +
		"limited by authorization checks on the metadata. \n" +
		"\n" +
		"Extreme caution should be taken in use of this system. \n" +
		"\n" +
		"There is no way to convert the system back to encrypted mode\n" +
		"without re-uploading files. \n" +
		"\n" +
		"Responses to all API calls will indicate that data is being \n" +
		"stored in an unencrypted format. This is to provide similar \n" +
		"warning to those users who would otherwise expect the data \n" +
		"to be encrypted based upon past experience using the service. \n" +
		"============================================================ \n" +
		"============================================================ \n" +
		"============================================================"
}
