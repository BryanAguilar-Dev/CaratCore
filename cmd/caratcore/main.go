package main

import (
	"fmt"
	"os"

	"github.com/BryanAguilar-Dev/CaratCore/internal/discovery"
)

func main() {
	if len(os.Args) > 2 {
		printUsage()
		os.Exit(2)
	}
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "doctor":
		case "help", "-h", "--help":
			printUsage()
			return
		default:
			printUsage()
			os.Exit(2)
		}
	}

	computer := discovery.Scan()

	fmt.Println("CaratCore")
	if computer.HostnameError != "" {
		fmt.Printf("Node: unavailable (%s)\n", computer.HostnameError)
	} else {
		fmt.Println("Node:", computer.Hostname)
	}
	fmt.Println("OS:", computer.OperatingSystem)
	if computer.Distribution != "" {
		fmt.Printf("Distribution: %s %s\n", computer.Distribution, computer.DistributionVersion)
	}
	fmt.Println("Architecture:", computer.Architecture)
	if computer.PlatformSupported {
		fmt.Println("Platform compatibility: supported (Debian 13 amd64)")
	} else {
		fmt.Println("Platform compatibility: unsupported (first target: Debian 13 amd64)")
	}
	if !computer.DockerCLIAvailable {
		fmt.Println("Docker CLI: not found")
		fmt.Println("Docker Engine: not checked")
	} else {
		fmt.Println("Docker CLI: available")
		if computer.DockerEngineAvailable {
			fmt.Printf("Docker Engine: ready (version %s)\n", computer.DockerVersion)
		} else {
			fmt.Printf("Docker Engine: unavailable (%s)\n", computer.DockerError)
		}
	}
}

func printUsage() {
	fmt.Println("Usage: caratcore [doctor | help]")
}
