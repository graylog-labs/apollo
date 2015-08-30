package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// TODO: multiple server instances? auto-discovery?

var username string
var password string
var url string

func init() {
	flag.StringVar(&username, "user", "", "Graylog username (must have administrator permissions)")
	flag.StringVar(&password, "password", "", "Graylog password")
	flag.StringVar(&url, "url", "", "URL of graylog-server REST URL. (Example: http://graylog.example.org:12900)")
}

func main() {
	// Parse and check CLI flags.
	flag.Parse()
	if !flagsSet() {
		flag.PrintDefaults()
		fmt.Print("\n")
		log.Fatal("Missing parameters. Exiting.")
	}

	// Set up logger.
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	log.Println("Starting up.")

	// Get all metrics and write to file.
	metrics := readMetrics()
	log.Printf("Successfully read %vkb of metrics from Graylog.", len(metrics)/1024)
	err := ioutil.WriteFile("metrics.json", metrics, 0644)
	check(err)

	// Zip up the files.
	zipIt(metrics)
	log.Println("Wrote bundle .zip file.")

	// Clean up tmp files.
	log.Println("Cleaning up temporary files.")
	os.Remove("metrics.json")

	log.Println("Finished.")
}

func flagsSet() bool {
	return len(username) > 0 && len(password) > 0 && len(url) > 0
}

func getHTTPRequest(path string) (*http.Client, *http.Request) {
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}

	req, err := http.NewRequest("GET", url+path, nil)

	if err != nil {
		check(err)
	}
	req.SetBasicAuth(username, password)

	client := &http.Client{}

	return client, req
}

func readMetrics() []byte {
	client, req := getHTTPRequest("system/metrics")

	resp, err := client.Do(req)

	if err != nil {
		check(err)
	}

	if resp.StatusCode != 200 {
		log.Fatal("Expected HTTP 200 but got HTTP " + strconv.Itoa(resp.StatusCode) + ". Exiting.")
	}

	result, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		check(err)
	}

	resp.Body.Close()

	return result
}

func zipIt(metrics []byte) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add files to the archive.
	var files = []struct {
		Name string
		Body []byte
	}{
		{"metrics.json", metrics},
	}
	for _, file := range files {
		zipFile, err := zipWriter.Create(file.Name)
		if err != nil {
			check(err)
		}
		_, err = zipFile.Write([]byte(file.Body))
		if err != nil {
			check(err)
		}
	}

	err := zipWriter.Close()
	check(err)

	// Write zipfile to disk.
	t := time.Now()
	finalName := fmt.Sprintf("graylog_apollo_bundle-%d-%02d-%02dT%02d-%02d-%02d.zip", t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	ioutil.WriteFile(finalName, buf.Bytes(), 0644) // TODO: include timestamp
}

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
