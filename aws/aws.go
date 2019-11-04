package aws

import (
	"net/http"
	"time"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sts"
)

type Config struct {
	Region   string
	Profile  string
	Endpoint string
}

func (ac *Config) Merge(otherCfg Config) {
	if ac.Region == "" {
		ac.Region = otherCfg.Region
	}

	if ac.Profile == "" {
		ac.Profile = otherCfg.Profile
	}

	if ac.Endpoint == "" {
		ac.Endpoint = otherCfg.Endpoint
	}
}

var awsPool = map[Config]*AWS{}

type AWS struct {
	CF              cloudformationiface.CloudFormationAPI
	S3UploadManager S3UploadManager
	AccountID       string
	Region          string
}

type Provider struct{}

func (p Provider) Must(cfg Config) *AWS {
	a, err := p.New(cfg)

	if err != nil {
		panic(err)
	}

	return a
}

func (Provider) New(cfg Config) (*AWS, error) {
	if aws, ok := awsPool[cfg]; ok {
		return aws, nil
	}

	sess := initSession(cfg)

	aws := AWS{}

	aws.CF = cloudformation.New(sess)
	aws.S3UploadManager = s3manager.NewUploader(sess)
	aws.Region = awssdk.StringValue(sess.Config.Region)

	callerIdent, err := sts.New(sess).GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return &aws, err
	}

	aws.AccountID = awssdk.StringValue(callerIdent.Account)

	awsPool[cfg] = &aws

	return &aws, nil
}

func initSession(cfg Config) *session.Session {
	opts := session.Options{}

	if cfg.Profile != "" {
		opts.Profile = cfg.Profile
	}

	awsCfg := awssdk.Config{}
	awsCfg.MaxRetries = awssdk.Int(7)

	if cfg.Region != "" {
		awsCfg.Region = awssdk.String(cfg.Region)
	}

	awsCfg.Endpoint = awssdk.String(cfg.Endpoint)

	httpClient := http.Client{
		Timeout: 2 * time.Second,
	}
	awsCfg.HTTPClient = &httpClient

	opts.Config = awsCfg
	opts.SharedConfigState = session.SharedConfigEnable

	return session.Must(session.NewSessionWithOptions(opts))
}

func nilString(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
