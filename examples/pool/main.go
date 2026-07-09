// Run with: go run ./examples/pool
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SimiyuWafulah/goconc/pool"
)

func main() {
	ctx := context.Background()
	p := pool.New(ctx, 3) // only 3 jobs run at once, no matter how many are submitted

	for i := 0; i < 10; i++ {
		i := i
		p.Submit(func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			fmt.Printf("job %d done\n", i)
			if i == 7 {
				return errors.New("job 7 failed on purpose")
			}
			return nil
		})
	}

	if err := p.Wait(); err != nil {
		fmt.Println("pool finished with error:", err)
		return
	}
	fmt.Println("all jobs finished successfully")
}
