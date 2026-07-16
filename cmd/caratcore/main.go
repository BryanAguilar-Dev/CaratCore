package main

import (
	"fmt"

	"github.com/BryanAguilar-Dev/CaratCore/internal/discovery"
)

func main() {

	computer := discovery.Scan()

	fmt.Println("CaratCore")
	fmt.Println("Node:", computer.Hostname)
}
