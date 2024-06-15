package main

import (
	"fmt"
	"log"
	"os"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
)

func main() {
	cfg := yamll.New("DEBUG", os.Args[1])
	cfg.SetLogger()

	yaml, err := cfg.Yaml()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(yaml)
}
