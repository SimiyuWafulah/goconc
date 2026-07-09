// Run with: go run ./examples/deadline
package main

import (
	"fmt"
	"time"

	"github.com/SimiyuWafulah/goconc/deadline"
)

func main() {
	m := deadline.NewMutex(200 * time.Millisecond)

	if err := m.Lock(); err != nil {
		panic(err)
	}
	fmt.Println("acquired the lock")

	// Simulate a second caller trying to acquire the same lock while it's
	// still held -- a plain sync.Mutex would just hang here forever.
	go func() {
		fmt.Println("second caller attempting to lock...")
		if err := m.Lock(); err != nil {
			fmt.Println("second caller gave up:", err)
			return
		}
		defer m.Unlock()
		fmt.Println("second caller got the lock (unexpected in this demo)")
	}()

	time.Sleep(400 * time.Millisecond) // long enough for the timeout above to fire
	m.Unlock()
	fmt.Println("first caller released the lock")
}
