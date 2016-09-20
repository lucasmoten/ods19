package server

import (
	"os"

	globalconfig "decipher.com/object-drive-server/config"
	configx "decipher.com/object-drive-server/configx"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/uber-go/zap"
)

// NewAWSSessionForS3 instantiates a connection to AWS.
func NewAWSSessionForS3(logger zap.Logger) *session.Session {
	//normally:  s3.awsamazon.com
	return newAWSSession(configx.OD_AWS_ENDPOINT, logger)
}

// NewAWSSessionForCW get a cloudwatch session
func NewAWSSessionForCW(logger zap.Logger) *session.Session {
	//normally: monitoring.us-east-1.awsamazon.com
	return newAWSSession(configx.OD_AWS_CLOUDWATCH_ENDPOINT, logger)
}

// NewAWSSession instantiates a connection to AWS.
func newAWSSession(service string, logger zap.Logger) *session.Session {

	configx.CheckAWSEnvironmentVars(logger)

	region := os.Getenv("AWS_REGION")
	endpoint := os.Getenv(service)

	// See if AWS creds in environment
	accessKeyID := globalconfig.GetEnvOrDefault(configx.OD_AWS_ACCESS_KEY_ID, globalconfig.GetEnvOrDefault("AWS_ACCESS_KEY_ID", ""))
	secretKey := globalconfig.GetEnvOrDefault(configx.OD_AWS_SECRET_ACCESS_KEY, globalconfig.GetEnvOrDefault("AWS_SECRET_ACCESS_KEY", ""))
	if len(accessKeyID) > 0 && len(secretKey) > 0 {
		logger.Info("aws.credentials", zap.String("provider", "environment variables"))
		var sessionConfig *aws.Config
		if len(endpoint) == 0 {
			sessionConfig = &aws.Config{
				Credentials: credentials.NewEnvCredentials(),
				Region:      aws.String(region),
			}
		} else {
			sessionConfig = &aws.Config{
				Credentials: credentials.NewEnvCredentials(),
				Region:      aws.String(region),
				Endpoint:    aws.String(endpoint),
			}
		}
		//sessionConfig = sessionConfig.WithLogLevel(aws.LogDebugWithHTTPBody).WithDisableComputeChecksums(false)
		return session.New(sessionConfig)
	}
	// Do as IAM
	logger.Info("aws.credentials", zap.String("provider", "iam role"))
	sessionConfig := &aws.Config{
		Region:   aws.String(region),
		Endpoint: aws.String(endpoint),
	}
	//sessionConfig = sessionConfig.WithLogLevel(aws.LogDebugWithHTTPBody).WithDisableComputeChecksums(false)
	return session.New(sessionConfig)
}
