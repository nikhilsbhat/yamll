package main

import (
	"fmt"
	"log"
	"os"

	"github.com/nikhilsbhat/yamll/pkg/yamll"
)

func main() {
	cfg := yamll.New(os.Args[1])

	yaml, err := cfg.Yaml()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(yaml)
}
