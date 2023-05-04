package main

import (
	"math/rand"
	"moxxiproxy/cmd"
	"time"
)

func main() {

	rand.Seed(time.Now().UnixMilli())
	cmd.Execute()
}
