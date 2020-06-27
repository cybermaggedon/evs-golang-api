package evs

import (
	"os"
	"strings"
)

type Config struct {
	Name    string
	Input   string
	Outputs []string
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
	}  else {
		c.SetOutputs(defout)
	}

	return c

}

func (c *Config) SetInput(val string) {
	c.Input = val
}

func (c *Config) SetOutputs(val []string) {
	c.Outputs = val
}
