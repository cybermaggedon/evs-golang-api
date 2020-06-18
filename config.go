
package cyberprobe

type Config struct {
	Input string
	Output []string
}

func NewConfig(defbind string) *evs.Config {

	c := &evs.Config{}

	if input, ok := os.LookupEnv("INPUT"); ok  {
		c.SetInput(input)
	} else {
		c.SetInput(defbind)
	}

	if output, ok := os.LookupEnv("OUTPUT"); ok {
		c.SetOutput(strings.Split(output, ","))
	}

	return c

}

func (c *Config) SetInput(val string) {
	c.Input = val
}

func (c *Config) SetOutput(val []string) {
	c.Output = val
}

