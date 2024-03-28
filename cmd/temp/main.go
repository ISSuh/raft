package main

import (
	"context"
)

func main() {
	c := context.Background()

	ctx, canel := context.WithCancel(c)
}
