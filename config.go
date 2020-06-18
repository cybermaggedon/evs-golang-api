package evs

import (
	"os"
	"strings"
)

type Config struct {
	Input   string
	Outputs []string
}

func NewConfig(defbind string) *Config {

	c := &Config{}

	if input, ok := os.LookupEnv("INPUT"); ok {
		c.SetInput(input)
	} else {
		c.SetInput(defbind)
	}

	if outputs, ok := os.LookupEnv("OUTPUT"); ok {
		c.SetOutputs(strings.Split(outputs, ","))
	}

	return c

}

func (c *Config) SetInput(val string) {
	c.Input = val
}

func (c *Config) SetOutputs(val []string) {
	c.Outputs = val
}
