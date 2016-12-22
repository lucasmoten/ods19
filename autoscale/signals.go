package autoscale

import (
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"decipher.com/object-drive-server/amazon"
	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/services/zookeeper"
	"github.com/aws/aws-sdk-go/aws"
	asg "github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/uber-go/zap"
)

const (
	//ASGTransitionTerminating signals that autoscaling is interested in shutting us down
	ASGTransitionTerminating = "autoscaling:EC2_INSTANCE_TERMINATING"
	//exitIgnore is a pseudo signal to return without killing the process.
	exitIgnore = 420
)

// AutoScaler is where all the interfaces for autoscaling related functionality reside
type AutoScaler struct {
	//Logger accepts messages as it works
	Logger zap.Logger
	//ZKState lets us signal to stop sending us work
	ZKState *zookeeper.ZKState
	//Config is our environment variables
	Config *config.AutoScalingConfig
	//SQS subset of the queue API we use
	SQS AutoScalerSQS
	//ASG subset of the queue API we use
	ASG AutoScalerASG
	//Sleep can be a null sleep
	Sleep func(t time.Duration)
	//ExitChannel gets our exit code here
	ExitChannel chan int
}

// AutoScalerSQS hides SQS
type AutoScalerSQS interface {
	ReceiveMessage(req *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(req *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
	GetQueueUrl(req *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error)
}

// AutoScalerASG hides ASG
type AutoScalerASG interface {
	RecordLifecycleActionHeartbeat(req *asg.RecordLifecycleActionHeartbeatInput) (*asg.RecordLifecycleActionHeartbeatOutput, error)
	CompleteLifecycleAction(req *asg.CompleteLifecycleActionInput) (*asg.CompleteLifecycleActionOutput, error)
}

//LifecycleMessage is related to us
type LifecycleMessage struct {
	AutoScalingGroupName string `json:"AutoScalingGroupName"`
	Service              string `json:"Service"`
	Time                 string `json:"Time"`
	AccountID            string `json:"AccountId"`
	LifecycleTransition  string `json:"LifecycleTransition"`
	RequestID            string `json:"RequestId"`
	LifecycleActionToken string `json:"LifecycleActionToken"`
	EC2InstanceID        string `json:"EC2InstanceId"`
	LifecycleHookName    string `json:"LifecycleHookName"`
}

// autoScalerTerminating lets autoscaling know that we are ready to have our instance terminated
func (as *AutoScaler) autoScalerTerminating(lifecycleMessage *LifecycleMessage) {
	actionResult := "CONTINUE"
	logger := as.Logger
	asgSession := as.ASG
	logger.Info("autoscale terminating")
	//Notify autoscaling that we are ready to be terminated
	if asgSession != nil && lifecycleMessage != nil {
		actionInput := &asg.CompleteLifecycleActionInput{
			AutoScalingGroupName:  aws.String(as.Config.AutoScalingGroupName),
			LifecycleActionResult: aws.String(actionResult),
			LifecycleActionToken:  aws.String(lifecycleMessage.LifecycleActionToken),
			LifecycleHookName:     aws.String(lifecycleMessage.LifecycleHookName),
		}
		_, err := asgSession.CompleteLifecycleAction(actionInput)
		if err != nil {
			logger.Warn("autoscale terminate fail", zap.String("err", err.Error()), zap.String("note", "ignore if this server was not spawned by autoscale"))
		} else {
			logger.Info("autoscale terminate success")
		}
	}
}

// waitingToTerminate tells autoscale to wait longer to terminate us
func (as *AutoScaler) waitingToTerminate(lifecycleMessage *LifecycleMessage, remainingFiles int) {
	logger := as.Logger.With(zap.Int("remainingFiles", remainingFiles))
	//If we have not terminated yet, then send a heartbeat
	if lifecycleMessage != nil {
		heartbeatInput := &asg.RecordLifecycleActionHeartbeatInput{
			AutoScalingGroupName: aws.String(as.Config.AutoScalingGroupName),
			LifecycleActionToken: aws.String(lifecycleMessage.LifecycleActionToken),
			LifecycleHookName:    aws.String(lifecycleMessage.LifecycleHookName),
		}
		if as.ASG != nil {
			_, err := as.ASG.RecordLifecycleActionHeartbeat(heartbeatInput)
			if err != nil {
				logger.Warn("autoscale heartbeat fail", zap.String("err", err.Error()), zap.String("note", "ignore if this server was not spawned by autoscale"))
			} else {
				logger.Info("autoscale heartbeat success")
			}
		}
	}
	//Wait for more files to evacuate
	logger.Info("autoscale waiting to terminate", zap.Int64("sleepInSeconds", as.Config.PollingInterval))
	as.Sleep(time.Duration(as.Config.PollingInterval) * time.Second)
}

// prepareForTermination is what we do in response to requests to shut down
func (as *AutoScaler) prepareForTermination(lifecycleMessage *LifecycleMessage) int {
	logger := as.Logger

	as.Logger.Info("prepare for termination")
	//Stop our zk connection to ensure that we have no more work left
	if as.ZKState != nil {
		zookeeper.ServiceStop(as.ZKState, "https", logger)
	}
	//Wait long enough that we are no longer getting new work
	as.Sleep(time.Duration(5 * time.Second))

	//Wait for our uploaded items to drop to zero
	if lifecycleMessage != nil {
		logger = logger.With(
			zap.String("AutoScalingGroupName", as.Config.AutoScalingGroupName),
			zap.String("LifecycleActionToken", lifecycleMessage.LifecycleActionToken),
			zap.String("LifecycleHookName", lifecycleMessage.LifecycleHookName),
		)
	}
	isChecking := true
	//Keep testing from going into an infinite loop.
	//It's unreasonable to wait forever in a real situation.
	tries := 10000
	for isChecking && tries > 0 {
		tries--
		if tries == 0 {
			logger.Info("autoscale termination giving up")
			return 1
		}
		//Wait for existing uploads to finish
		remainingFiles := 0
		caches := ciphertext.FindCiphertextCacheList()
		for _, dp := range caches {
			remainingFiles += dp.CountUploaded()
		}
		if remainingFiles == 0 {
			//
			// No more checking required, as we have evacuated files safely
			//
			isChecking = false
			as.autoScalerTerminating(lifecycleMessage)
		} else {
			//
			//  We are waiting for uploads into S3 to complete
			//
			as.waitingToTerminate(lifecycleMessage, remainingFiles)
		}
	}
	logger.Info("exited")
	return 0
}

// handleLifecycleMessage handles shutdown message from autoscaling
// Log lifecycle transition messages that pertain to us
func (as *AutoScaler) handleLifecycleMessage(m string, warn bool) *LifecycleMessage {
	logger := as.Logger
	if warn {
		logger.Info("sqs message", zap.String("val", m))
	}
	var parsed LifecycleMessage
	err := json.Unmarshal([]byte(m), &parsed)
	//Assume that this queue is ONLY used for lifecycle messages for this scaling group.
	if warn {
		logger.Warn("sqs unparseable", zap.String("err", err.Error()))
	}
	//These messages can be anything - xml, json, etc -, so if we use these simple struct parses, we just need
	//to ignore the errors completely
	if parsed.LifecycleTransition != "" {
		//If it's our machine
		if strings.Compare(parsed.EC2InstanceID, as.Config.EC2InstanceID) == 0 {
			logger.Info(
				"sqs autoscaling lifecycle transition observed",
				zap.Object("parsed", parsed),
			)
			return &parsed
		}
	}
	return nil
}

// watchForShutdownBySignals is invoked by the OS.  No message parsing or notification is required.
// This won't terminate without a signal.  Don't call this directly.
func (as *AutoScaler) watchForShutdownBySignals() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1)

	for sig := range sigchan {
		switch sig {
		case syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2:
			as.ExitChannel <- as.prepareForTermination(nil)
		default:
			as.Logger.Info("signal hard shutdown")
			as.ExitChannel <- 1
		}
	}
}

// WatchForShutdownByMessage watches for SQS messages to terminate
// - we MUST send a signal back from this function
func (as *AutoScaler) WatchForShutdownByMessage() {
	logger := as.Logger

	if len(as.Config.QueueName) == 0 {
		logger.Info("sqs queue is configured to be turned off")
		return
	}

	//Log setup parameters as sanity check
	logger.Info(
		"sqs queue",
		zap.String("queueName", as.Config.QueueName),
		zap.String("autoScaleGroup", as.Config.AutoScalingGroupName),
		zap.String("ec2InstanceID", as.Config.EC2InstanceID),
	)

	//Get the first batch before entering the loop so that we can check
	//for existence of the queue.  This lets us have a "null" implementation by
	//just not setting it up.
	var sqsRcv *sqs.ReceiveMessageOutput

	//Get the name of the queue - we MUST do this from the API
	queueURLOutput, err := as.SQS.GetQueueUrl(
		&sqs.GetQueueUrlInput{
			QueueName: aws.String(as.Config.QueueName),
		},
	)
	if err != nil {
		logger.Error(
			"sqs queue get url failed",
			zap.String("err", err.Error()),
		)
		as.ExitChannel <- exitIgnore
		return
	}

	//Start receiving messages
	logger.Info(
		"sqs queue url",
		zap.String("url", *queueURLOutput.QueueUrl),
	)

	//Get the first batch
	sqsRcv, err = as.SQS.ReceiveMessage(
		&sqs.ReceiveMessageInput{
			QueueUrl:            queueURLOutput.QueueUrl,
			MaxNumberOfMessages: &(as.Config.QueueBatchSize),
		},
	)
	if err != nil {
		logger.Error("sqs queue error",
			zap.String("err", err.Error()),
			zap.Object("config", as.Config),
		)
		as.ExitChannel <- exitIgnore
		return
	}

	logger.Info("sqs queried for messages")
	for {
		messagesForUs := 0
		//check the messages
		if sqsRcv != nil && err == nil {
			for _, m := range sqsRcv.Messages {
				if m != nil && m.Body != nil {
					ourMessage := as.handleLifecycleMessage(*m.Body, false)
					if ourMessage != nil {
						as.SQS.DeleteMessage(
							&sqs.DeleteMessageInput{
								QueueUrl:      queueURLOutput.QueueUrl,
								ReceiptHandle: m.ReceiptHandle,
							},
						)
						messagesForUs++
						logger.Info("sqs deleted our message")
						if ourMessage.LifecycleTransition == ASGTransitionTerminating {
							as.ExitChannel <- as.prepareForTermination(ourMessage)
							return
						}
					}
				}
			}
		}

		//Wait before getting more messages from the queue if we got zero messages for us
		if messagesForUs == 0 {
			as.Sleep(time.Duration(as.Config.PollingInterval) * time.Second)
		}

		//Get more messages
		sqsRcv, err = as.SQS.ReceiveMessage(
			&sqs.ReceiveMessageInput{
				QueueUrl:            queueURLOutput.QueueUrl,
				MaxNumberOfMessages: &(as.Config.QueueBatchSize),
			},
		)
		if err != nil {
			logger.Error("sqs rcv fail", zap.String("err", err.Error()))
		}
	}
}

// WatchForShutdown looks for requests to shut down, either through signals or messages
func WatchForShutdown(z *zookeeper.ZKState, logger zap.Logger) {

	//Get an SQS session
	sqsConfig := config.NewAutoScalingConfig()

	//We use this to get remote messages to shut down
	var sqsSession AutoScalerSQS = sqs.New(amazon.NewAWSSession(sqsConfig.AWSConfigSQS, logger))
	//We need this if we want to stay alive after getting a termination message.
	var asgSession AutoScalerASG = asg.New(amazon.NewAWSSession(sqsConfig.AWSConfigASG, logger))

	as := &AutoScaler{
		Logger:      logger,
		ZKState:     z,
		Config:      sqsConfig,
		SQS:         sqsSession,
		ASG:         asgSession,
		Sleep:       time.Sleep,
		ExitChannel: make(chan int, 1),
	}

	//The normal signal path - it just exits without a signal so that we don't need another exit channel
	go as.watchForShutdownBySignals()

	//The thread that does the same thing in response to a queue message
	go as.WatchForShutdownByMessage()

	//We must have this thread to read the other side of the queue
	go func() {
		//Take the first real exit that is sent to our channel
		for {
			result := <-as.ExitChannel
			//exitIgnore is incomplete signal handling setup that logs an error without taking down the server.
			if result != exitIgnore {
				os.Exit(result)
			}
		}
	}()
}
