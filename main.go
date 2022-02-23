package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type sorting struct {
	key int
	val string
}

type sorted []sorting

func (s sorted) Len() int           { return len(s) }
func (s sorted) Less(i, j int) bool { return s[i].val < s[j].val }
func (s sorted) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func main() {
	// set and read config
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.SetConfigName("config")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	// get parameters
	minutes := flag.String("t", "", "user input last n minutes")
	directory := flag.String("d", "", "user input log directory")
	flag.Parse()

	// check minute parameter
	var numCheck = regexp.MustCompile(`^[0-9]+$`)
	*minutes = strings.Replace(*minutes, "m", "", -1)
	var minutesCheck = numCheck.MatchString(*minutes)

	if *minutes == "" || minutesCheck == false {
		fmt.Println("Please fill/check parameter: -t for last n minutes, eg. 10m")
		os.Exit(2)
	}

	// check directory parameter
	if *directory == "" {
		fmt.Println("Please fill parameter: -d for log directory")
		os.Exit(2)
	}

	// read directory
	files, err := ioutil.ReadDir(*directory)
	if err != nil {
		log.Fatal(err)
	}

	readFile := map[int]string{}
	for k, file := range files {
		readFile[k] = file.Name()
	}

	// sort read files from descending because of the newest is the lastest :
	// "The logs and log files are stored sequentially, i.e. from oldest to newest"
	var fileSorted sorted
	for k, v := range readFile {
		fileSorted = append(fileSorted, sorting{key: k, val: v})
	}

	sort.Sort(sort.Reverse(fileSorted))

	// set timezone in UTC based on sample log :
	// 127.0.0.1 user-identifier frank [10/Oct/2017:13:54:00 +0000] "GET/api/endpoint HTTP/1.0" 200 5134
	location, _ := time.LoadLocation("UTC")

	// set now and n last time for time compare
	now := time.Now().In(location)

	count, err := strconv.Atoi(*minutes)
	if err != nil {
		log.Fatal(err)
	}
	timeCompare := now.Add(time.Duration(-count) * time.Minute)

	// open log files
	for _, v := range fileSorted {
		file, err := os.Open(*directory + "/" + v.val)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		found := 0
		scanner := bufio.NewScanner(file)

		// scan log rows
		for scanner.Scan() {
			split := strings.Split(scanner.Text(), " ")
			timeLog, _ := time.Parse("02/Jan/2006:15:04:05", strings.Replace(split[3], "[", "", -1))

			if timeLog.After(timeCompare) && timeLog.Before(now) && strings.Contains(scanner.Text(), "HTTP/1.0\" "+viper.GetString("search.httpCode")) {
				fmt.Println(scanner.Text())
				found++
			}
		}

		if found == 0 {
			break
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
}
