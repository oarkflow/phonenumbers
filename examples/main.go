package main

import (
	"fmt"
	
	"github.com/oarkflow/phonenumbers"
)

func main() {
	err := phonenumbers.LoadNetworks()
	if err != nil {
		panic(err)
	}
	fmt.Println(phonenumbers.Verify("9856034616", "NP"))
}
