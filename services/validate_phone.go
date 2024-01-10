package services

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/oarkflow/phonenumbers"
)

func GetCsvHeader(scanner *bufio.Scanner, comma rune) map[int]string {
	scanner.Scan()
	r := csv.NewReader(strings.NewReader(scanner.Text()))
	r.Comma = comma
	r.TrimLeadingSpace = true
	colHeader, _ := r.Read()
	colPosition := make(map[int]string)
	for key, col := range colHeader {
		colPosition[key] = col
	}
	return colPosition
}

func ValidatePhoneReader(reader io.Reader, phoneKey string, comma rune, defaultPrefix string) []map[string]string {
	scanner := bufio.NewScanner(reader)
	colPosition := GetCsvHeader(scanner, comma)
	for k, v := range colPosition {
		colPosition[k] = clean([]byte(v))
	}
	jobs := make(chan []byte)
	results := make(chan map[string]string)

	wg := new(sync.WaitGroup)
	for w := 1; w <= 2; w++ {
		wg.Add(1)
		go ProcessNumber(jobs, results, wg, colPosition, phoneKey, comma, defaultPrefix)
	}
	go func() {
		for scanner.Scan() {
			jobs <- scanner.Bytes()
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var data []map[string]string
	for v := range results {
		data = append(data, v)
	}

	return data
}

func ValidatePhone(csvFile, out, phoneKey string, comma rune, outputComma rune, defaultPrefix string) error {
	file, err := os.Open(csvFile)
	if err != nil {
		return errors.New("File not found or unable to open")
	}
	outFile, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return errors.New("File not found")
	}
	defer outFile.Close()
	scanner := bufio.NewScanner(file)
	colPosition := GetCsvHeader(scanner, comma)
	for k, v := range colPosition {
		colPosition[k] = clean([]byte(v))
	}
	jobs := make(chan []byte)
	results := make(chan map[string]string)

	wg := new(sync.WaitGroup)
	for w := 1; w <= 2; w++ {
		wg.Add(1)
		go ProcessNumber(jobs, results, wg, colPosition, phoneKey, comma, defaultPrefix)
	}
	go func() {
		for scanner.Scan() {
			jobs <- scanner.Bytes()
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	writer := csv.NewWriter(outFile)
	writer.Comma = outputComma
	defer writer.Flush()
	header := make(map[int]string)

	count := 0
	for v := range results {
		h := make([]string, len(v))
		d := make([]string, len(v))
		idx := 0
		if count == 0 {
			for col, _ := range v {
				header[idx] = col
				h[idx] = col
				idx++
			}
			err = writer.Write(h)
			if err != nil {
				panic(err)
			}
		}

		for col, val := range v {
			for id, head := range header {
				if col == head {
					d[id] = val
				}
			}
		}

		err = writer.Write(d)
		if err != nil {
			panic(err)
		}
		count++
	}

	return nil
}

func clean(s []byte) string {
	j := 0
	for _, b := range s {
		if ('a' <= b && b <= 'z') ||
			('A' <= b && b <= 'Z') ||
			('0' <= b && b <= '9') ||
			b == ' ' || b == '_' {
			s[j] = b
			j++
		}
	}
	return strings.TrimSpace(string(s[:j]))
}

func ProcessNumber(jobs <-chan []byte, results chan<- map[string]string, wg *sync.WaitGroup, col map[int]string, phoneKey string, comma rune, defaultPrefix string) {
	// Decreasing internal counter for wait-group as soon as goroutine finishes
	defer wg.Done()

	// eventually I want to have a []string channel to work on a chunk of lines not just one line of text
	for j := range jobs {
		data := make(map[string]string)
		r := csv.NewReader(bytes.NewReader(j))
		r.Comma = comma
		r.TrimLeadingSpace = true
		fields, _ := r.Read()
		for key, dt := range fields {
			data[col[key]] = strings.TrimSpace(dt)
		}
		num := phonenumbers.Number{DefaultPrefix: defaultPrefix}
		num.Phone = strings.TrimSpace(data[phoneKey])
		num.Verify()
		validatedPhone := "validated_" + phoneKey
		validPhone := "invalid_" + phoneKey
		data["region"] = num.CountryCode
		data[validatedPhone] = num.Phone
		data["phone_type_label"] = num.PhoneTypeHuman
		data["carrier_name"] = num.CarrierName
		data["carrier_mnc"] = num.CarrierMnc
		data["carrier_mcc"] = num.CarrierMcc
		data["carrier_nnc"] = num.CarrierNnc
		data["phone_type_code"] = fmt.Sprintf("%d", num.PhoneType)
		data["dial_code"] = fmt.Sprintf("%d", num.DialCode)
		data[validPhone] = fmt.Sprintf("%v", num.Invalid)
		results <- data
	}
}
