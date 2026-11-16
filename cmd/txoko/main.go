// Command txoko is a cozy corner for your terminal that grows as you work.
package main

import (
	"os"

	"github.com/LifeOfAitor/txoko/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
