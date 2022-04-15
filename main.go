package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

var (
	input     = flag.String("input", "", "input file")
	snakecase = flag.Bool("snakecase", false, "whether to convert fields to snake_case")
)

func init() {
	flag.Parse()

	if *input == "" {
		log.Fatal("input is required")
	}
}

func main() {
	f, err := excelize.OpenFile(*input)

	if err != nil {
		log.Fatal(err)
	}

	sheets := f.GetSheetList()

	wg := new(sync.WaitGroup)

	for _, sheet := range sheets {
		wg.Add(1)

		go func(sheet string) {
			defer wg.Done()

			start := time.Now()

			log.Printf("Processing sheet %s\n", sheet)

			rows, err := f.Rows(sheet)

			if err != nil {
				log.Printf("Encountered an error while processing sheet %s : %s\n", sheet, err)
				return
			}

			count := 0

			var header []string
			var data [][]string

			log.Printf("Reading rows from sheet %s\n", sheet)

			for rows.Next() {
				count++

				row, err := rows.Columns()

				if err != nil {
					log.Printf("Encountered an error while reading row %d from sheet %s : %s\n", count, sheet, err)
					return
				}

				switch {
				case count == 1:
					header = row
				case count > 1:
					data = append(data, row)
				default:
				}
			}

			log.Printf("Read %d rows from sheet %s\n", count, sheet)

			if *snakecase {
				log.Printf("Converting fields to snake_case\n")

				re := regexp.MustCompile(`\s+`)

				for i, field := range header {
					header[i] = strings.ToLower(string(re.ReplaceAll([]byte(field), []byte("_"))))
				}
			}

			b := new(strings.Builder)

			indent := "    "

			b.WriteString("[\n")

			for _, datum := range data {
				m := make(map[string]string)

				for i, field := range header {
					m[field] = datum[i]
				}

				bytes, err := json.MarshalIndent(m, indent, indent)

				if err != nil {
					log.Printf("Encountered an error while marshalling row %d from sheet %s : %s\n", count, sheet, err)
					return
				}

				b.WriteString(indent + string(bytes) + ",\n")
			}

			b.WriteString("]")

			filename := sheet + ".json"

			if err = ioutil.WriteFile(filename, []byte(b.String()), 0777); err != nil {
				log.Printf("Encountered an error while writing %s for sheet %s : %s\n", filename, sheet, err)
				return
			}

			log.Printf("Wrote %s for sheet %s, took %f seconds\n", filename, sheet, time.Now().Sub(start).Seconds())
		}(sheet)
	}

	wg.Wait()
}
