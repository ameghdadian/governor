package governor_test

import (
	"sync"
	"testing"

	"github.com/ameghdadian/governor"
)

type counter struct {
	c int
}

func TestChanSync(t *testing.T) {
	ss, cleanup := governor.New(counter{c: 10})
	defer cleanup()

	var wg sync.WaitGroup

	numGoRoutine := 200
	wg.Add(numGoRoutine)

	for range numGoRoutine {
		go func() {
			defer wg.Done()
			ss.Write(func(i *counter) {
				i.c += 10
			})
		}()
	}

	wg.Wait()
	v, err := ss.Read()
	if err != nil {
		t.Fatal("cannot read from channel:", err)
	}
	if v.c != 2010 {
		t.Fatal("got unexpected result:", v.c)
	}
}
