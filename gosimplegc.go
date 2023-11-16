package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

var WeekdayLang = []string{"niedziela", "poniedziałek", "wtorek", "środa", "czwartek", "piątek", "sobota"}

type t_outout struct {
	Timestamp    string `json:"timestamp"`
	Next_date    string `json:"next_date"`
	Days         int    `json:"days"`
	Weekday      int    `json:"weekday"`
	Weekday_name string `json:"weekday_name"`
	Value        int    `json:"value"`
}

var LOG *log.Logger
var local_location *time.Location
var DEBUG = os.Getenv("DEBUG") == "1"

func _find_nearest_date(data map[int][]int, month int, curr_date time.Time, next_time *time.Time, next_days *int) {
	var dest_date time.Time
	year, _month, _ := curr_date.Date()
	__month := int(_month)
	if __month > month {
		year += 1
	}

	for _, day := range data[month] {
		dest_date = time.Date(year, time.Month(month), day, 0, 0, 0, 0, local_location)

		diff := int(dest_date.Sub(curr_date).Hours() / 24)

		LOG.Printf(" ++ DATE %s -> %s = %d", curr_date.Format("2006-01-02"), dest_date.Format("2006-01-02"), diff)

		if diff >= 0 && diff < *next_days {
			*next_days = diff
			*next_time = dest_date
		}
	}
}

func find_nearest_date(data map[int][]int, curr_date time.Time) (time.Time, int) {
	var next_days int = 999
	var next_time time.Time

	_, _month, _ := curr_date.Date()

	month := int(_month)

	_find_nearest_date(data, month, curr_date, &next_time, &next_days)

	if next_days < 999 {
		LOG.Printf("Found next day: %s (in %d days)\n", next_time, next_days)
		return next_time, next_days
	}

	month = month + 1
	if month > 12 {
		month = 1
	}

	_find_nearest_date(data, month, curr_date, &next_time, &next_days)

	if next_days < 999 {
		LOG.Printf("Found next day: %s (in %d days)\n", next_time, next_days)
		return next_time, next_days
	}

	return next_time, -1
}

func load_yaml(input_file string, dest *map[interface{}]interface{}) bool {
	LOG.Printf("Input file: %s\n", input_file)

	file, err := os.Open(input_file)
	if err != nil {
		LOG.Fatal("error: ", err)
	}

	defer func() {
		if err = file.Close(); err != nil {
			LOG.Fatal("error: ", err)
			return // false
		}
	}()

	input_text, err := ioutil.ReadAll(file)
	if err != nil {
		return false
	}

	err = yaml.Unmarshal(input_text, &dest)
	if err != nil {
		LOG.Fatal("error: ", err)
	}

	if err != nil {
		LOG.Fatal("error: ", err)
	}

	return true
}

func main() {
	LOG = log.New(os.Stderr, "* ", 0)

	// curr_date := time.Date(2023, 12, 15, 0, 0, 0, 0, time.Now().Location())
	curr_date := time.Now()
	local_location = curr_date.Location()

	var input_file string = ""
	var input_data map[interface{}]interface{}
	var garbage_type string = ""
	var garbage_type_set bool = false
	var ok bool

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		LOG.Fatal("Usage: gosimplegc config [selector]")

	}

	if len(args) > 1 {
		garbage_type = args[1]
		garbage_type_set = true
	}

	if !DEBUG {
		LOG.SetOutput(ioutil.Discard)
	}

	input_file = args[0]

	LOG.Printf("Current date: %s", curr_date)
	LOG.Printf("Read file: %s", input_file)
	LOG.Printf("Garbage type: %s\n", garbage_type)

	if input_file == "" {
		LOG.Fatal("Empty file name")
	}

	input_data = make(map[interface{}]interface{})
	ok = load_yaml(input_file, &input_data)

	if !ok {
		LOG.Fatal("Failed to load and parse input yaml")
	}

	var name string = ""
	var output_path string = ""

	for k := range input_data {
		name = k.(string)
		output_path = ""

		if garbage_type_set && garbage_type != name {
			LOG.Printf("Skip: %s", name)
			continue
		}

		mm := input_data[k].(map[string]interface{})
		// LOG.Printf("break 1: %s\n", mm)

		output_path, ok = mm["output"].(string)
		if !ok {
			LOG.Print("No output_path")
		}

		_bymonth := mm["bymonth"]

		// LOG.Printf(" + _bymonth: %s\n", _bymonth)

		bymonth, ok := _bymonth.(map[interface{}]interface{})
		if !ok {
			LOG.Printf("Failed to read bymonth: %s", _bymonth)
			os.Exit(1)
		}

		var data map[int][]int
		var days []int

		data = make(map[int][]int)

		for _month, _days := range bymonth {
			month := _month.(int)
			days = []int{}

			for _, d := range _days.([]interface{}) {
				days = append(days, d.(int))
			}
			data[month] = days
		}

		var next_time, next_days = find_nearest_date(data, curr_date)
		weekday := int(next_time.Weekday())

		output := t_outout{Timestamp: curr_date.Format(time.RFC3339), Next_date: next_time.Format("2006-01-02"), Weekday: weekday, Weekday_name: WeekdayLang[weekday], Days: next_days, Value: 2}
		if next_days == 1 {
			output.Value = 1
		} else if next_days == 0 {
			output.Value = 0
		}

		output_json, err := json.Marshal(output)
		if err != nil {
			LOG.Println("Falied to create output json: %s", err)
		}

		if (output_path == "") || (output_path == "-") {
			fmt.Printf("%s\n", output_json)
			continue
		}

		err = os.WriteFile(output_path, output_json, 0666)
		if err != nil {
			LOG.Fatal("error: ", err)
		}

	}
}
