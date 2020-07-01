package evs

import (
	"fmt"
	"os"
	"strings"
)

var (

	// Location of broker service
	default_broker = "pulsar://localhost:6650"

	// Pulsar topic persistency, should be persistent or non-persistent
	default_persistence = "non-persistent"

	// Tenant, default is public, only relevant in a multi-tenant deployment
	default_tenant = "public"

	// Namespace, default is the default.
	default_namespace = "default"
)

type Config struct {
	Broker      string
	Name        string
	Input       string
	Outputs     []string
	Persistence string
	Tenant      string
	Namespace   string
}

func NewConfig(defname, defbind string, defout []string) *Config {

	c := &Config{}

	if name, ok := os.LookupEnv("ANALYTIC_NAME"); ok {
		c.Name = name
	} else {
		c.Name = defname
	}

	if input, ok := os.LookupEnv("INPUT"); ok {
		c.SetInput(input)
	} else {
		c.SetInput(defbind)
	}

	if outputs, ok := os.LookupEnv("OUTPUT"); ok {
		c.SetOutputs(strings.Split(outputs, ","))
	} else {
		c.SetOutputs(defout)
	}

	if val, ok := os.LookupEnv("PULSAR_BROKER"); ok {
		c.Broker = val
	} else {
		c.Broker = default_broker
	}

	if val, ok := os.LookupEnv("PULSAR_PERSISTENCE"); ok {
		c.SetPersistence(val)
	} else {
		c.SetPersistence(default_persistence)
	}

	if val, ok := os.LookupEnv("PULSAR_TENANT"); ok {
		c.SetTenant(val)
	} else {
		c.SetTenant(default_tenant)
	}

	if val, ok := os.LookupEnv("PULSAR_NAMESPACE"); ok {
		c.SetNamespace(val)
	} else {
		c.SetNamespace(default_namespace)
	}

	return c

}

func (c *Config) SetInput(val string) {
	c.Input = val
}

func (c *Config) SetOutputs(val []string) {
	c.Outputs = val
}

func (c *Config) SetPersistence(val string) {
	c.Persistence = val
}

func (c *Config) SetNamespace(val string) {
	c.Namespace = val
}

func (c *Config) SetTenant(val string) {
	c.Tenant = val
}

func (c *Config) GetInputTopic() string {
	return fmt.Sprintf("%s://%s/%s/%s", c.Persistence, c.Tenant,
		c.Namespace, c.Input)
}

type HasInputTopics interface {
	GetName() string
	GetInputTopic() string
}

type HasOutputTopics interface {
	GetName() string
	GetOutputTopics() []string
}

func (c *Config) GetOutputTopics() []string {
	out := make([]string, len(c.Outputs))
	for i, v := range c.Outputs {
		out[i] = fmt.Sprintf("%s://%s/%s/%s", c.Persistence, c.Tenant,
			c.Namespace, v)
	}
	return out
}

func (c *Config) GetName() string {
	return c.Name
}
