package autoscale_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/autoscale"
	"bitbucket.di2e.net/dime/object-drive-server/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	ASGTransitionInitializing = "autoscaling:EC2_INSTANCE_INITIALIZING"
	ASGTransitionNone         = ""
)

// Mock out SQS
type testAutoScalerSQS struct {
	titles      []string
	rawMessages []autoScalingTestResult
	messages    *sqs.ReceiveMessageOutput
	expected    []string
	deleted     []bool
	index       int
	t           *testing.T
}

type autoScalingTestResult struct {
	Entry    string
	Input    string
	Expected string
}

//mock out SQS - here is a queue
func newTestAutoScalerSQS(t *testing.T) *testAutoScalerSQS {
	rawMessages := []autoScalingTestResult{
		autoScalingTestResult{"Not even JSON", `testing`, ASGTransitionNone},
		autoScalingTestResult{"Correct initializing", `{  
          "AutoScalingGroupName":"defaultScalingGroup",
          "Service":"AWS Auto Scaling",
          "Time":"2016-02-26T21:09:59.517Z",
          "AccountId":"some-account-id",
          "LifecycleTransition":"autoscaling:EC2_INSTANCE_INITIALIZING",
          "RequestId":"some-request-id-2",
          "LifecycleActionToken":"some-token",
          "EC2InstanceId":"defaultec2",
          "LifecycleHookName":"graceful_shutdown_asg"
        }`, ASGTransitionInitializing},
		autoScalingTestResult{"Another machine initializing", `{  
          "AutoScalingGroupName":"defaultScalingGroup",
          "Service":"AWS Auto Scaling",
          "Time":"2016-02-26T21:09:59.517Z",
          "AccountId":"some-account-id",
          "LifecycleTransition":"autoscaling:EC2_INSTANCE_INITIALIZING",
          "RequestId":"some-request-id-2",
          "LifecycleActionToken":"some-token",
          "EC2InstanceId":"defaultec2A",
          "LifecycleHookName":"graceful_shutdown_asg"
        }`, ASGTransitionNone},
		autoScalingTestResult{"Json unrelated", `{
          "fark":"blah"
        }`, ASGTransitionNone},
		autoScalingTestResult{"Malformed scaling", `{
          "AutoScalingGroupName":"defaultScalingGroup",
        }`, ASGTransitionNone},
		autoScalingTestResult{"Correct termination", `{  
          "AutoScalingGroupName":"defaultScalingGroup",
          "Service":"AWS Auto Scaling",
          "Time":"2016-02-26T21:09:59.517Z",
          "AccountId":"some-account-id",
          "LifecycleTransition":"autoscaling:EC2_INSTANCE_TERMINATING",
          "RequestId":"some-request-id-2",
          "LifecycleActionToken":"some-token",
          "EC2InstanceId":"defaultec2",
          "LifecycleHookName":"graceful_shutdown_asg"
        }`, autoscale.ASGTransitionTerminating},
	}
	as := testAutoScalerSQS{}
	as.rawMessages = rawMessages
	count := len(rawMessages)
	as.titles = make([]string, count)
	as.messages = &sqs.ReceiveMessageOutput{
		Messages: make([]*sqs.Message, count),
	}
	as.expected = make([]string, count)
	as.deleted = make([]bool, count)
	for i := range rawMessages {
		as.titles[i] = rawMessages[i].Entry
		as.messages.Messages[i] = &sqs.Message{
			Body:          aws.String(rawMessages[i].Input),
			ReceiptHandle: aws.String(fmt.Sprintf("%d", i)),
		}
		as.expected[i] = rawMessages[i].Expected
	}
	as.t = t
	return &as
}

func (s *testAutoScalerSQS) ReceiveMessage(req *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	//Just return them all in first chunk
	s.t.Logf("recv messages")
	if s.index == 0 {
		s.index++
		return s.messages, nil
	}
	return nil, nil
}

func (s *testAutoScalerSQS) DeleteMessage(req *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	//Make sure that we delete our messages (only)
	idx, err := strconv.Atoi(*req.ReceiptHandle)
	if err != nil {
		s.t.Fail()
	}
	s.deleted[idx] = true
	s.t.Logf("deleted %d", idx)
	//We won't read again, so just ignore deletes
	return nil, nil
}

func (s *testAutoScalerSQS) GetQueueUrl(req *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	return &sqs.GetQueueUrlOutput{
		QueueUrl: aws.String("https://sqs.us-east-1.amazonaws.com"),
	}, nil
}

type testAutoScalerASG struct {
	t *testing.T
}

func newTestAutoScalerASG(t *testing.T) *testAutoScalerASG {
	return &testAutoScalerASG{t: t}
}

func (s *testAutoScalerASG) RecordLifecycleActionHeartbeat(req *autoscaling.RecordLifecycleActionHeartbeatInput) (*autoscaling.RecordLifecycleActionHeartbeatOutput, error) {
	return nil, nil
}

func (s *testAutoScalerASG) CompleteLifecycleAction(req *autoscaling.CompleteLifecycleActionInput) (*autoscaling.CompleteLifecycleActionOutput, error) {
	return nil, nil
}

func TestAutoScale(t *testing.T) {
	logger := config.RootLogger
	//Simulate a queue of messages, and make sure that we parse what we need to see, and ignore what we do not
	sqs := newTestAutoScalerSQS(t)
	as := &autoscale.AutoScaler{
		Logger:      logger,
		Config:      config.NewAutoScalingConfig(),
		SQS:         sqs,
		ASG:         newTestAutoScalerASG(t),
		ExitChannel: make(chan int, 1),
	}

	//Cause the queue messages to match up, no matter how our environment is configured.
	as.Config.AutoScalingGroupName = "defaultScalingGroup"
	as.Config.EC2InstanceID = "defaultec2"
	as.Config.QueueName = "defaultLifecycleQueue"
	as.Sleep = func(t time.Duration) {
		//Use a fake sleep to make the test not take forever
		logger.Info("sleep fake ... zzz")
	}

	//Go through the motions of getting and parsing a message.
	go as.WatchForShutdownByMessage()
	exitCode := <-as.ExitChannel

	for i := range sqs.rawMessages {
		expected := sqs.rawMessages[i].Expected
		deleted := sqs.deleted[i]
		isOurMessage := expected != ""

		if isOurMessage && !deleted {
			//We must delete our messages
			t.Logf("we did not delete our own message '%s' %d", expected, i)
			t.FailNow()
		}
		if !isOurMessage && deleted {
			//Dont delete the messages of others
			t.Logf("we deleted somebody elses message %d", i)
			t.FailNow()
		}
	}
	//If we did not finish, then something is wrong
	if exitCode != 0 {
		t.Logf("not finished")
		t.FailNow()
	} else {
		t.Logf("autoscale test finished correctly")
	}
}
