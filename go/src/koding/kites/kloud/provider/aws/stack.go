package oldaws

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"koding/kites/kloud/api/amazon"
	"koding/kites/kloud/stack"
	"koding/kites/kloud/stackplan"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

const arnPrefix = "arn:aws:iam::"

// Cred represents jCredentialDatas.meta for "aws" provider.
type Cred struct {
	Region    string `json:"region" bson:"region" hcl:"region"`
	AccessKey string `json:"access_key" bson:"access_key" hcl:"access_key"`
	SecretKey string `json:"secret_key" bson:"secret_key" hcl:"secret_key"`
}

var _ stack.Validator = (*Cred)(nil)

type Bootstrap struct {
	ACL       string `json:"acl" bson:"acl" hcl:"acl"`
	CidrBlock string `json:"cidr_block" bson:"cidr_block" hcl:"cidr_block"`
	IGW       string `json:"igw" bson:"igw" hcl:"igw"`
	KeyPair   string `json:"key_pair" bson:"key_pair" hcl:"key_pair"`
	RTB       string `json:"rtb" bson:"rtb" hcl:"rtb"`
	SG        string `json:"sg" bson:"sg" hcl:"sg"`
	Subnet    string `json:"subnet" bson:"subnet" hcl:"subnet"`
	VPC       string `json:"vpc" bson:"vpc" hcl:"vpc"`
	AMI       string `json:"ami" bson:"ami" hcl:"ami"`
}

var _ stack.Validator = (*Bootstrap)(nil)

func (b *Bootstrap) Valid() error {
	if b.ACL == "" {
		return errors.New(`bootstrap value for "acl" is empty`)
	}
	if b.CidrBlock == "" {
		return errors.New(`bootstrap value for "cidr_block" is empty`)
	}
	if b.IGW == "" {
		return errors.New(`bootstrap value for "igw" is empty`)
	}
	if b.KeyPair == "" {
		return errors.New(`bootstrap value for "key_pair" is empty`)
	}
	if b.RTB == "" {
		return errors.New(`bootstrap value for "rtb" is empty`)
	}
	if b.SG == "" {
		return errors.New(`bootstrap value for "sg" is empty`)
	}
	if b.Subnet == "" {
		return errors.New(`bootstrap value for "subnet" is empty`)
	}
	if b.VPC == "" {
		return errors.New(`bootstrap value for "vpc" is empty`)
	}
	if b.AMI == "" {
		return errors.New(`bootstrap value for "ami" is empty`)
	}
	return nil

	var meta Meta
}

// Credentials creates new AWS credentials value from the given meta.
func (c *Cred) Credentials() *credentials.Credentials {
	return credentials.NewStaticCredentials(c.AccessKey, c.SecretKey, "")
}

// Options creates new amazon client options.
func (c *Cred) Options() *amazon.ClientOptions {
	return &amazon.ClientOptions{
		Credentials: c.Credentials(),
		Region:      c.Region,
	}
}

func (c *Cred) session() *session.Session {
	return amazon.NewSession(c.Options())
}

// AccountID parses an AWS arn string to get the Account ID.
func (c *Cred) AccountID() (string, error) {
	user, err := iam.New(c.session()).GetUser(nil)
	if err == nil {
		return parseAccountID(aws.StringValue(user.User.Arn))
	}

	for msg := err.Error(); msg != ""; {
		i := strings.Index(msg, arnPrefix)

		if i == -1 {
			break
		}

		msg = msg[i:]

		accountID, e := parseAccountID(msg)
		if e != nil {
			continue
		}

		return accountID, nil
	}

	return "", err
}

// The function assumes arn string comes from an IAM resource, as
// it treats region empty.
//
// For details see:
//
//   http://docs.aws.amazon.com/IAM/latest/UserGuide/reference_identifiers.html#identifiers-arns
//
// Example arn string: "arn:aws:iam::213456789:user/username"
// Returns: 213456789
func parseAccountID(arn string) (string, error) {
	if !strings.HasPrefix(arn, arnPrefix) {
		return "", fmt.Errorf("invalid ARN: %q", arn)
	}

	accountID := arn[len(arnPrefix):]
	i := strings.IndexRune(accountID, ':')

	if i == -1 {
		return "", fmt.Errorf("invalid ARN: %q", arn)
	}

	accountID = accountID[:i]

	if accountID == "" {
		return "", fmt.Errorf("invalid ARN: %q", arn)
	}

	return accountID, nil
}

// Valid implements the kloud.Validator interface.
func (meta *Cred) Valid() error {
	if meta.Region == "" {
		return errors.New("aws meta: region is empty")
	}
	if meta.AccessKey == "" {
		return errors.New("aws meta: access key is empty")
	}
	if meta.SecretKey == "" {
		return errors.New("aws meta: secret key is empty")
	}
	return nil
}

// Stack implements the kloud.StackProvider interface.
type Stack struct {
	*stackplan.BaseStack
}

var _ stackplan.Stack = (*Stack)(nil)

func (s *Stack) Credential() *Cred {
	return m.BaseStack.Credential.(*Cred)
}

func (s *Stack) Bootstrap() *Bootstrap {
	return m.BaseMachine.Bootstrap.(*Bootstrap)
}

// VerifyCredential
func (s *Stack) VerifyCredential(credential interface{}) error {
	return nil
}

// BootstrapTemplates
func (s *Stack) BootstrapTemplates() ([]*stack.Template, error) {
	return nil, nil
}

// BuildResources
func (s *Stack) BuildResources() error {
	return nil
}

func (s *Stack) BuildMetadata(m *stackplan.Machine) interface{} {
	meta := &Meta{
		Region:           s.Credential().Region,
		InstanceID:       m.Attributes["id"],
		AvailabilityZone: m.Attributes["availability_zone"],
		PlacementGroup:   m.Attributes["placement_group"],
	}

	if n, err := strconv.Atoi(m.Attributes["root_block_device.0.volume_size"]); err == nil {
		meta.StorageSize = n
	}

	return meta
}
