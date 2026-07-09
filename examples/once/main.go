// Run with: go run ./examples/once
package main

import (
	"errors"
	"fmt"

	"github.com/SimiyuWafulah/goconc/once"
)

func main() {
	var g once.Group

	g.Go(func() error {
		fmt.Println("fetching user profile...")
		return nil
	})
	g.Go(func() error {
		fmt.Println("fetching billing info...")
		return errors.New("billing service unavailable")
	})
	g.Go(func() error {
		fmt.Println("fetching preferences...")
		return nil
	})

	if err := g.Wait(); err != nil {
		fmt.Println("one of the fetches failed:", err)
		return
	}
	fmt.Println("all fetches succeeded")
}
