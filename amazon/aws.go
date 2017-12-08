package amazon

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/deciphernow/object-drive-server/config"
	"go.uber.org/zap"
)

// NewAWSSession instantiates a connection to AWS.
//
// Information on how to use the returned session can be found in the
// AWS API, https://docs.aws.amazon.com/sdk-for-go/api/aws/session.
func NewAWSSession(awsConfig *config.AWSConfig, logger zap.Logger) *session.Session {
	theCredentials := credentials.NewStaticCredentials(awsConfig.AccessKeyID, awsConfig.SecretAccessKey, "")
	if len(awsConfig.AccessKeyID) > 0 && len(awsConfig.SecretAccessKey) > 0 {
		logger.Info("aws.credentials", zap.String("provider", "environment variables"))
		var sessionConfig *aws.Config
		if len(awsConfig.Endpoint) == 0 {
			sessionConfig = &aws.Config{
				Credentials: theCredentials,
				Region:      aws.String(awsConfig.Region),
			}
		} else {
			sessionConfig = &aws.Config{
				Credentials: theCredentials,
				Region:      aws.String(awsConfig.Region),
				Endpoint:    aws.String(awsConfig.Endpoint),
			}
		}
		return session.New(sessionConfig)
	}
	// Do as IAM
	logger.Info("aws.credentials", zap.String("provider", "iam role"))
	sessionConfig := &aws.Config{
		Region:   aws.String(awsConfig.Region),
		Endpoint: aws.String(awsConfig.Endpoint),
	}
	return session.New(sessionConfig)
}
