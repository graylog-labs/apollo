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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/howeyc/gopass"
)

var (
	username       string
	password       string
	passwordPrompt bool
	url            string
	submitToken    string
)

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

type IndexRangesList struct {
	Ranges []IndexRangeDetails `json:"ranges"`
}

type IndexRangeDetails struct {
	IndexName    string `json:"index_name"`
	Begin        string `json:"begin"`
	End          string `json:"end"`
	CalculatedAt string `json:"calculated_at"`
	TookMs       int    `json:"took_ms"`
}

func init() {
	flag.StringVar(&username, "user", "", "Graylog username (must have administrator permissions)")
	flag.BoolVar(&passwordPrompt, "password", false, "Prompt for a Graylog password")
	flag.StringVar(&url, "url", "", "URL of a graylog-server REST URL. (Example: http://graylog.example.org:12900)")
}

func haveRequiredInput() []error {
	errs := make([]error, 0)
	if len(username) == 0 {
		errs = append(errs, fmt.Errorf("Username isn't set, please use `-user $USERNAME` or set the environmental variable `GRAYLOG_USER`"))
	}

	if len(password) == 0 {
		errs = append(errs, fmt.Errorf("Password isn't set, please use `-password` to prompt for the password or set the environmental variable `GRAYLOG_PASSWORD`"))
	}
	if len(url) == 0 {
		errs = append(errs, fmt.Errorf("URL isn't set, please use `-url $URL` or set the environmental variable `GRAYLOG_URL`"))
	}

	return errs
}

func main() {
	// Parse and check the enviornmental variables.
	for _, e := range os.Environ() {
		kv := strings.SplitN(e, "=", 2)

		switch kv[0] {
		case "GRAYLOG_USER":
			username = kv[1]
		case "GRAYLOG_PASSWORD":
			password = kv[1]
		case "GRAYLOG_URL":
			url = kv[1]
		}
	}

	// Parse and check CLI flags
	flag.Parse()

	if passwordPrompt {
		fmt.Printf("Graylog Password: ")
		password = string(gopass.GetPasswd())
	}

	if errs := haveRequiredInput(); len(errs) != 0 {
		fmt.Println("Unable to start the collector because:")
		for _, err := range errs {
			fmt.Println("\tError: ", err)
		}
		os.Exit(1)
	}

	// Set up logger.
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	log.Println("Starting up.")

	// Discovery. Read system detail information of all nodes.
	nodesListResponse := readResourceJson("system/cluster/nodes")
	var discoveredNodes ClusterNodeList
	err := json.Unmarshal(nodesListResponse, &discoveredNodes)
	check(err)

	var files []IncludedFile

	// Write bundle meta information files.
	var t = time.Now()
	files = append(files, IncludedFile{"timestamp", []byte(t.UTC().Format(time.RFC3339))})
	files = append(files, IncludedFile{"reporting_system.json", readResourceJson("system")})

	// Data that is the same, no matter requested from which graylog-server node.
	files = append(files, IncludedFile{"cluster_nodes.json", nodesListResponse})
	files = append(files, IncludedFile{"cluster_stats.json", readResourceJson("system/cluster/stats")})
	files = append(files, IncludedFile{"notifications.json", readResourceJson("system/notifications")})
	files = append(files, IncludedFile{"streams.json", readResourceJson("streams")})
	files = append(files, IncludedFile{"indexer_health.json", readResourceJson("system/indexer/cluster/health")})
	files = append(files, IncludedFile{"indexer_failures.json", readResourceJson("system/indexer/failures?limit=100&offset=0")})

	// Iterate over all discovered nodes.
	for i := 0; i < len(discoveredNodes.Nodes); i++ {
		node := discoveredNodes.Nodes[i]
		log.Printf("Discovered Graylog node: [%v] at [%v].\n", node.NodeId, node.TransportAddress)

		// Data specific to the requested graylog-server node. TODO: use method to unify calls
		files = append(files, IncludedFile{node.NodeId + "-system.json", readResourceJsonFromNode(node.TransportAddress, "system")})
		files = append(files, IncludedFile{node.NodeId + "-metrics.json", readResourceJsonFromNode(node.TransportAddress, "system/metrics")})
		files = append(files, IncludedFile{node.NodeId + "-system_jvm.json", readResourceJsonFromNode(node.TransportAddress, "system/jvm")})
		files = append(files, IncludedFile{node.NodeId + "-system_stats.json", readResourceJsonFromNode(node.TransportAddress, "system/stats")})
		files = append(files, IncludedFile{node.NodeId + "-services.json", readResourceJsonFromNode(node.TransportAddress, "system/serviceManager")})
		files = append(files, IncludedFile{node.NodeId + "-journal.json", readResourceJsonFromNode(node.TransportAddress, "system/journal")})
		files = append(files, IncludedFile{node.NodeId + "-buffers.json", readResourceJsonFromNode(node.TransportAddress, "system/buffers")})
		files = append(files, IncludedFile{node.NodeId + "-throughput.json", readResourceJsonFromNode(node.TransportAddress, "system/throughput")})
		files = append(files, IncludedFile{node.NodeId + "-system_messages.json", readResourceJsonFromNode(node.TransportAddress, "system/messages")})

		// Needs at least Graylog v1.3
		if nodeHasResource(node.TransportAddress, "system/loggers/messages/recent") {
			files = append(files, IncludedFile{node.NodeId + "-log.json", readResourceJsonFromNode(node.TransportAddress, "system/loggers/messages/recent?limit=500")})
		}
	}

	// Get all index ranges.
	indexRangesResponse := readResourceJson("system/indices/ranges")
	var indexRanges IndexRangesList
	err = json.Unmarshal(indexRangesResponse, &indexRanges)
	check(err)

	// Read and store shard routing for each index range.
	for i := 0; i < len(indexRanges.Ranges); i++ {
		indexRange := indexRanges.Ranges[i]
		files = append(files, IncludedFile{"indexrouting-" + indexRange.IndexName + ".json", readResourceJson("system/indexer/indices/" + indexRange.IndexName)})
	}

	filename := zipIt(files)
	log.Printf("Wrote bundle to file: %v\n", filename)

	log.Println("Finished.")
}

func getHTTPRequest(targetUrl string, path string) (*http.Client, *http.Request) {
	if !strings.HasSuffix(targetUrl, "/") {
		targetUrl += "/"
	}

	req, err := http.NewRequest("GET", targetUrl+path, nil)

	if err != nil {
		check(err)
	}
	req.SetBasicAuth(username, password)

	client := &http.Client{
		Timeout: time.Duration(30 * time.Second),
	}

	return client, req
}

func nodeHasResource(node string, path string) bool {
	client, req := getHTTPRequest(node, path)

	resp, err := client.Do(req)

	if err != nil {
		check(err)
	}

	return resp.StatusCode == 200
}

func readResourceJsonFromNode(node string, path string) []byte {
	client, req := getHTTPRequest(node, path)

	resp, err := client.Do(req)

	if err != nil {
		check(err)
	}

	if resp.StatusCode != 200 {
		log.Printf("Expected HTTP <200> but got HTTP <%v> at [%v].\n", strconv.Itoa(resp.StatusCode), path)

		if resp.StatusCode == 401 {
			log.Fatal("POSSIBLE CAUSE: Make sure that you are running this with a Graylog user that has admin permissions.")
		}

		log.Fatal("Exiting with failure.")
	}

	result, err := ioutil.ReadAll(resp.Body)
	check(err)

	resp.Body.Close()

	log.Printf("Successfully read %v bytes [%v].", len(result), path)

	return result
}

func readResourceJson(path string) []byte {
	return readResourceJsonFromNode(url, path)
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
