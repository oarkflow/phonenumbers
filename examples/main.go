package main

import (
	"fmt"

	"github.com/oarkflow/phonenumbers"
)

func main() {
	fmt.Println(phonenumbers.Verify("9856034616", "NP"))
}
