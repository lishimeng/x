package broker

import (
	"context"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	wait := make(chan int)
	c := context.Background()
	ctx, cancel := context.WithCancel(c)
	time.AfterFunc(time.Second, func() {
		cancel()
		wait <- 1
	})
	Ins = New(ctx)
	t.Logf("%v", Ins)
	<-wait
}
