package marathon

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"koding/kites/kloud/stack"
	"koding/kites/kloud/stack/provider"

	marathon "github.com/gambol99/go-marathon"
)

var schema = &provider.Schema{
	NewCredential: func() interface{} {
		return &Credential{}
	},
	NewBootstrap: nil,
	NewMetadata: func(m *stack.Machine) interface{} {
		if m == nil {
			return &Metadata{}
		}

		meta := &Metadata{
			AppID: m.Attributes["app_id"],
		}

		if n, err := strconv.Atoi(m.Attributes["app_count"]); err == nil {
			meta.AppCount = n
		}

		return meta
	},
}

var (
	_ stack.Validator = (*Credential)(nil)
	_ stack.Validator = (*Metadata)(nil)
)

// Credential represents credential information
// that are required to deploy a Marathon app.
type Credential struct {
	URL                string `json:"url" bson:"url" hcl:"url"`
	BasicAuthUser      string `json:"basic_auth_user" bson:"basic_auth_user" hcl:"basic_auth_user"`
	BasicAuthPassowrd  string `json:"basic_auth_password" bson:"basic_auth_password" hcl:"basic_auth_password"`
	RequestTimeout     int    `json:"request_timeout" bson:"request_timeout" hcl:"request_timeout"`
	DeploymentTimepout int    `json:"deployment_timeout" bson:"deployment_timeout" hcl:"deployment_timeout"`
}

// Valid implements the stack.Validator interface.
func (c *Credential) Valid() error {
	if c.URL == "" {
		return errors.New("invalid empty URL")
	}

	if _, err := url.Parse(c.URL); err != nil {
		return errors.New("invalid URL: " + err.Error())
	}

	return nil
}

// Config gives new configuration for Marathon client.
func (c *Credential) Config() *marathon.Config {
	cfg := marathon.NewDefaultConfig()

	cfg.URL = c.URL
	cfg.HTTPBasicAuthUser = c.BasicAuthUser
	cfg.HTTPBasicPassword = c.BasicAuthPassowrd
	cfg.HTTPClient = &http.Client{
		Timeout: c.requestTimeout() * time.Second,
	}

	return &cfg
}

func (c *Credential) requestTimeout() time.Duration {
	if c.RequestTimeout != 0 {
		return time.Duration(c.RequestTimeout)
	}

	return 600
}

// Metadata represents a single app metadata.
type Metadata struct {
	AppID    string  `json:"app_id" bson:"app_id" hcl:"app_id"`
	AppCount int     `json:"app_count" bson:"app_count" hcl:"app_count"`
	CPU      float64 `json:"cpu" bson:"cpu" hcl:"cpu"`
	Mem      float64 `json:"mem" bson:"mem" hcl:"mem"`
}

// Valid implements the stack.Validator interface.
func (m *Metadata) Valid() error {
	if m.AppID == "" {
		return errors.New("invalid empty app ID or group name")
	}

	return nil
}
