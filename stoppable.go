
package evs

import (
	"context"
	"os/signal"
	"os"
	"syscall"
	"log"
)

type Stoppable interface {
	Stop()
}

type Interruptible struct {
	Context      context.Context
	Cancel       context.CancelFunc
	stop    Stoppable
}

func (i *Interruptible) RegisterStop(stop Stoppable) {

	i.stop = stop

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <- c:
			log.Print("Signal recevied")
			i.stop.Stop()
		}
	}()

	i.Context, i.Cancel = context.WithCancel(context.Background())

}

