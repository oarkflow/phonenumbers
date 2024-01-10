package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/oarkflow/phonenumbers"
	"github.com/oarkflow/phonenumbers/services"
)

func main() {
	err := phonenumbers.LoadNetworks()
	if err != nil {
		panic(err)
	}
	bulkPhoneFromCsv()
}

func singlePhone() {
	fmt.Println(phonenumbers.Verify("9856034616", "NP"))
}

func bulkPhoneFromCsv() {
	data := `id ,phone,first_name,last_name
1 ,9856034616,"Sujit", "Baniya"
2 ,9805832689,"Anita", "Baniya"
`
	reader := strings.NewReader(data)
	rs := services.ValidatePhoneReader(reader, "phone", ',', "NP")
	fmt.Println(rs)
}

func cleanCsv() {
	data := `id,phone,first_name, last_name
1,1234567,"John", "Kumar Das "
2,123123,"Michelle", "Dan"`
	r := csv.NewReader(strings.NewReader(data))
	r.TrimLeadingSpace = true
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(record)
	}
}
