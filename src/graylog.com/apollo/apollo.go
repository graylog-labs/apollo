package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TODO:
//       can we get cluster_id?
//       make sure that there never is alarmcallback or output config in streams
//       read extractors
//       if code 401, run with admin permissions
//       sample threaddumps
//       get shards and shit. are there unassigned or red ones/

var username string
var password string
var url string
var submitToken string

type IncludedFile struct {
	Name string
	Body []byte
}

type ClusterNodeList struct {
	Nodes []ClusterNodeDetails `json:"nodes"`
}

type ClusterNodeDetails struct {
	NodeId           string `json:"node_id"`
	Type             string `json:"type"`
	TransportAddress string `json:"transport_address"`
	LastSeen         string `json:"last_seen"`
	ShortNodeId      string `json:"short_node_id"`
	IsMaster         bool   `json:"is_master"`
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

	// Discovery. Read system detail information of all nodes.
	nodesListResponse := readResourceJson("system/cluster/nodes", true)
	var discoveredNodes ClusterNodeList
	err := json.Unmarshal(nodesListResponse, &discoveredNodes)
	check(err, true)

	var discoveredClusterNodeDetails string
	for i := 0; i < len(discoveredNodes.Nodes); i++ {
		node := discoveredNodes.Nodes[i]
		log.Printf("Discovered Graylog node: [%v] at [%v].\n", node.NodeId, node.TransportAddress)

		// Try to read /system from discovered node. Do not exit it it fails.
		discoveredClusterNodeDetails += string(readResourceJsonFromNode(node.TransportAddress, "system", false))
		discoveredClusterNodeDetails += "\n\n"

		// Build own JSON file with all /system informations. Backend can list all nodes and see which were discovered
		// - required in case one node is not reachable (overloaded?) but we want as much information as possible.
	}

	// Zip up the files.
	var files = []IncludedFile{
		{"reporting_system.json", readResourceJson("system", true)},
		{"metrics.json", readResourceJson("system/metrics", true)},
		{"system_jvm.json", readResourceJson("system/jvm", true)},
		{"system_stats.json", readResourceJson("system/stats", true)},
		{"cluster_stats.json", readResourceJson("system/cluster/stats", true)},
		{"cluster_nodes.json", nodesListResponse},
		{"cluster_nodes_details.json", []byte(discoveredClusterNodeDetails)},
		{"services.json", readResourceJson("system/serviceManager", true)},
		{"journal.json", readResourceJson("system/journal", true)},
		{"buffers.json", readResourceJson("system/buffers", true)},
		{"notifications.json", readResourceJson("system/notifications", true)},
		{"throughput.json", readResourceJson("system/throughput", true)},
		{"streams.json", readResourceJson("streams", true)},
		{"streams_throughput.json", readResourceJson("streams/throughput", true)},
		{"indexer_health.json", readResourceJson("system/indexer/cluster/health", true)},
		{"indexer_failures.json", readResourceJson("system/indexer/failures?limit=100&offset=0", true)},
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

func getHTTPRequest(targetUrl string, path string, fail bool) (*http.Client, *http.Request) {
	if !strings.HasSuffix(targetUrl, "/") {
		targetUrl += "/"
	}

	req, err := http.NewRequest("GET", targetUrl+path, nil)

	if err != nil {
		check(err, fail)
	}
	req.SetBasicAuth(username, password)

	client := &http.Client{}

	return client, req
}

func readResourceJsonFromNode(node string, path string, fail bool) []byte {
	client, req := getHTTPRequest(node, path, fail)

	resp, err := client.Do(req)

	if err != nil {
		check(err, fail)
	}

	if resp.StatusCode != 200 {
		log.Println("Expected HTTP 200 but got HTTP " + strconv.Itoa(resp.StatusCode) + ".")

		if fail {
			log.Fatal("Exiting.")
		}
	}

	result, err := ioutil.ReadAll(resp.Body)
	check(err, fail)

	resp.Body.Close()

	log.Printf("Successfully read %v bytes [%v].", len(result), path)

	return result
}

func readResourceJson(path string, fail bool) []byte {
	return readResourceJsonFromNode(url, path, fail)
}

func zipIt(files []IncludedFile) string {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, file := range files {
		zipFile, err := zipWriter.Create(file.Name)
		if err != nil {
			check(err, true)
		}
		_, err = zipFile.Write([]byte(file.Body))
		if err != nil {
			check(err, true)
		}
	}

	err := zipWriter.Close()
	check(err, true)

	// Write zipfile to disk.
	t := time.Now()
	finalName := fmt.Sprintf("graylog_apollo_bundle-%d-%02d-%02dT%02d-%02d-%02d.zip", t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	ioutil.WriteFile(finalName, buf.Bytes(), 0644)

	return finalName
}

func check(e error, fail bool) {
	if e != nil {
		if fail {
			log.Fatal(e)
		} else {
			log.Println(e)
		}
	}
}
