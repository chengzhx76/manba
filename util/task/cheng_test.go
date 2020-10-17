package task

import (
	"testing"
	"time"
)

func Test_task(t *testing.T) {

	//runner := NewRunner()
	NewRunner()
	/*c := make(chan struct{})
	defer close(c)

	err := runner.RunTask(func() {
		c <- struct{}{}
	})

	if err != nil {
		t.Error("run task failed, return a error", err)
		return
	}

	select {
	case <-c:
	case <-time.After(time.Millisecond * 50):
		t.Error("run task failed, task not run after 50ms")
	}*/

	time.Sleep(1 * time.Hour)

}
