package main

import (
	"fmt"
	"os"
)

const file = "/operator-resources.yaml"

func main() {
	buf, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(buf))
}
