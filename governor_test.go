package governor_test

import (
	"governor"
	"sync"
	"testing"
)

type Counter struct {
	C int
}

func TestChanSync(t *testing.T) {
	c, cleanup := governor.New(Counter{C: 10})
	defer cleanup()

	var wg sync.WaitGroup

	numGoRoutine := 200
	wg.Add(numGoRoutine)

	for range numGoRoutine {
		go func() {
			defer wg.Done()
			c.Write(func(i *Counter) {
				i.C += 10
			})
		}()
	}

	wg.Wait()
	v, err := c.Read()
	if err != nil {
		t.Fatal("cannot read from channel:", err)
	}
	if v.C != 2010 {
		t.Fatal("got unexpected result:", v.C)
	}
}
