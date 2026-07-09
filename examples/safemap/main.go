// Run with: go run ./examples/safemap
package main

import (
	"fmt"
	"sync"

	"github.com/SimiyuWafulah/goconc/safemap"
)

func main() {
	m := safemap.New[string, int]()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("worker-%d", n%5)
			m.Set(key, n)
		}(i)
	}
	wg.Wait()

	fmt.Println("entries after concurrent writes:")
	m.Range(func(k string, v int) bool {
		fmt.Printf("  %s = %d\n", k, v)
		return true
	})
	fmt.Println("total keys:", m.Len())
}
