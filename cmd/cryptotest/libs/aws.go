package libs

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func generateSession(account string) *s3.S3 {
	sessionConfig := &aws.Config{
		Credentials: credentials.NewSharedCredentials("", account),
	}
	sess := session.New(sessionConfig)
	svc := s3.New(sess)
	return svc
}
