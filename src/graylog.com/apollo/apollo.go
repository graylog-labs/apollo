package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TODO: multiple server instances? auto-discovery? required with cluster stats read we already do?
//       collect info about cpu usage, cores, io, ...? is there a lib for that? do we have it in rest?
//       can we get cluster_id?
//       make sure that there never is alarmcallback or output config in streams

var username string
var password string
var url string
var submitToken string

type IncludedFile struct {
	Name string
	Body []byte
}

func init() {
	flag.StringVar(&username, "user", "", "Graylog username (must have administrator permissions)")
	flag.StringVar(&password, "password", "", "Graylog password")
	flag.StringVar(&url, "url", "", "URL of graylog-server REST URL. (Example: http://graylog.example.org:12900)")
	flag.StringVar(&submitToken, "submit", "", "Provie an Apollo token to submit directly. (optional)")
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

	// Zip up the files.
	var files = []IncludedFile{
		{"metrics.json", readResourceJson("system/metrics")},
		{"system.json", readResourceJson("system")},
		{"system_jvm.json", readResourceJson("system/jvm")},
		{"system_stats.json", readResourceJson("system/stats")},
		{"cluster_stats.json", readResourceJson("system/cluster/stats")},
		{"cluster_nodes.json", readResourceJson("system/cluster/nodes")},
		{"services.json", readResourceJson("system/serviceManager")},
		{"journal.json", readResourceJson("system/journal")},
		{"buffers.json", readResourceJson("system/buffers")},
		{"notifications.json", readResourceJson("system/notifications")},
		{"throughput.json", readResourceJson("system/throughput")},
		{"streams.json", readResourceJson("streams")},
		{"streams_throughput.json", readResourceJson("streams/throughput")},
	}

	filename := zipIt(files)
	log.Printf("Wrote bundle to file: %v\n", filename)

	// Submit to Apollo directly?
	if len(submitToken) > 0 {
		log.Printf("Submission to Apollo service was requested. Submitting with token [%v].\n", submitToken)

		// TODO: actually build this.
	}

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

func readResourceJson(path string) []byte {
	client, req := getHTTPRequest(path)

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

	log.Printf("Successfully read %v bytes [%v] from Graylog.", len(result), path)

	return result
}

func zipIt(files []IncludedFile) string {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

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
	ioutil.WriteFile(finalName, buf.Bytes(), 0644)

	return finalName
}

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
