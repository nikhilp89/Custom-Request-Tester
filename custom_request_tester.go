package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pieterclaerhout/go-waitgroup"
	"gopkg.in/yaml.v3"
)

var (
	requestsFile string
	subdomainsFile string
	grepLocation string
	grepHeader string
	grepString string
	grepStatusCode int

	requestsFileData = map[string]Request{}
	subdomains = []string{}

	matchedSubdomains = []string{}

	proxyStr = "http://localhost:8080"

	requestMethod string
	requestUrl string
	requestProtocol string
	requestBody string

	customHeaders = map[string]string{
		"Accept": "*/*",
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.93 Safari/537.36",
	}
)
type fileStruct struct {
	Mu    sync.Mutex
	Data []string
	FileName string
}

type Request struct {
	Method string
	Url string
	Protocol string
	Headers []Header
	Body string

}

type Header struct {
	Name string
	Value string
}

/*
Initialiatizes input command line argument variables
*/
func init() {
	flag.StringVar(&requestsFile, "headers", "", "first string")
	flag.StringVar(&subdomainsFile, "subdomains", "", "second string")
	flag.StringVar(&grepLocation, "grepLocation", "default", "third string")
	flag.StringVar(&grepHeader, "grepHeader", "default", "fourth string")
	flag.StringVar(&grepString, "grep", "test", "fifth string")
	flag.IntVar(&grepStatusCode, "grepStatusCode", 0, "sixth string")
}

/*
Checking if all necessary command line arguments have been supplied
*/
func checkArguments() {
	if (len(requestsFile) == 0) || (len(subdomainsFile) == 0) {
		log.Println("Usage: go run main.go -headers string_value -subdomains string_value")
		flag.PrintDefaults()
		os.Exit(1)
	}
}

/*
Creates a new output file to store query results
*/
func newFile() *fileStruct {
	_, err := os.Stat("output")

	if err != nil {
		_, err := os.Create("output")
		if err != nil {
			log.Printf("File Error: %s", err)
		}
	}

	newFileStruct := fileStruct{Mu: sync.Mutex{}, Data: []string{}, FileName: "output"}

	return &newFileStruct
}

/*
Write string supplied to output file. Access regulated through a mutex.
*/
func (j *fileStruct) WriteToFile(data string) {
	j.Mu.Lock()
	defer j.Mu.Unlock()

	j.Data = append(j.Data, data)

	var f *os.File

	f, err := os.Create(j.FileName)
	if err != nil {
		log.Printf("Weire File Error: %s", err)
	}

	defer f.Close()

	for _, value := range j.Data {
		f.WriteString(value)
	}
}

/*
Scans an input file and returns the lines in a string slice.
*/
func scanTextInputFile(fileName string) []string {

	var output []string

	f, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("File Reading Error: %v", err)
	}

	s := bufio.NewScanner(f)

	for s.Scan() {
		output = append(output, s.Text())
	}

	return output
}

/*
Scans an input yaml file.
*/
func scanYamlInputFile(fileName string) map[string]Request {

	yamlFile, err := ioutil.ReadFile(fileName)

	if err != nil {

        log.Fatal("Yaml Error: ", err)
    }

	data := make(map[string]Request)

	err2 := yaml.Unmarshal(yamlFile, &data)

	if err2 != nil {

    	log.Fatal("Yaml Error: ", err2)
    }

	return data
}

/*
Go through contents of request YAML and populate headers, cookies, ...
*/
func parseRequestYaml(requestValues Request) {

	requestMethod = requestValues.Method
	requestUrl = requestValues.Url
	requestProtocol = requestValues.Protocol

	for _, header := range requestValues.Headers {
		customHeaders[header.Name] = header.Value
	}

	requestBody = requestValues.Body

	fmt.Println("requestMethod: ", requestMethod)
	fmt.Println("requestUrl: ", requestUrl)
	fmt.Println("requestProtocol: ", requestProtocol)

	fmt.Println("requestHeaders: ")

	for n, v := range customHeaders {
		fmt.Println(n, ": ", v)
	}

	fmt.Println("requestBody: ", requestBody)
}

/*
Checking the response body, headers or both for the string specified by the user.
*/
func performGrep(uri string, res *http.Response, outputFile *fileStruct) {

	defer res.Body.Close()
		
	body, error := ioutil.ReadAll(res.Body)

	if error != nil {
		log.Println("Grep Error: ", error)
	}

	reg, _ := regexp.Compile(grepString) 

	if grepLocation == "default" {
		log.Printf("%s: %d", uri, res.StatusCode)
		log.Printf("%s: %s", uri, res.Header.Get("Content-Type"))
		outputFile.WriteToFile("default")
		outputFile.WriteToFile("default1")

	} else if grepLocation == "body" {

		if reg.MatchString(string(body)) {
			outputFile.WriteToFile(uri + "\n")
		}

	} else if grepLocation == "headers" {

		//if no specific header was specified, scan all headers
		if grepHeader == "default" {
			
			for headerName, _ := range res.Header {

				if reg.MatchString(strings.ToLower(res.Header.Get(headerName))) && (res.StatusCode == grepStatusCode || grepStatusCode == 0) {
					outputFile.WriteToFile(uri + "\n")
				}
			}
		} else { //A specific header was specified

			_, exists := res.Header[grepHeader]

			if exists {

				log.Println("Header Exists")

				if reg.MatchString(strings.ToLower(res.Header.Get(grepHeader))) && (res.StatusCode == grepStatusCode || grepStatusCode == 0) {
					outputFile.WriteToFile(uri + "| " + res.Header.Get(grepHeader) + "\n")
				}
			} else {
				log.Println("Header Does Not Exist")
			}
		}
	} else if grepLocation == "statuscode" {
		if res.StatusCode == grepStatusCode {
			outputFile.WriteToFile(uri + "\n")
		}
	}
}

func generateRequest(uri string, headers map[string]string, outputFile *fileStruct) {

	log.Printf("Testing: %s", uri)

	tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns: 1000,
		MaxIdleConnsPerHost: 1000,
    	}

	var netClient = &http.Client {
		Transport: tr,
		Timeout: time.Second * 3,
	}

	req, err := http.NewRequest(requestMethod, uri, nil)
	if err != nil {
		log.Println("Error making request: ", err)
	} else {
		for name, value := range headers {
			
			req.Header.Add(name,value)
		}

		res, err := netClient.Do(req)
		if err != nil {
			log.Printf("Error sending request: %v", err)
		} else {
			performGrep(uri, res, outputFile)
		}
	}
}

func generateRequest1(uri string, headers map[string]string, outputFile *fileStruct) {

	log.Printf("Testing: %s", uri)

	tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns: 1000,
		MaxIdleConnsPerHost: 1000,
    	}

	var netClient = &http.Client {
		Transport: tr,
		Timeout: time.Second * 10,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var i int

	for i < 100 {

		req, err := http.NewRequest(requestMethod, uri, nil)
		if err != nil {
			log.Fatal("Error making request: ", err)
		}

		for name, value := range headers {
			
			req.Header.Add(name,value)
		}

		res, err := netClient.Do(req)
		if err != nil {
			log.Printf("Error sending request: %v", err)
		} else {
			if(res.StatusCode == 200) {
				performGrep(uri, res, outputFile)
				break
			} else {
				performGrep(uri, res, outputFile)

				uri = res.Header.Get("Location")
			}			
		}

		i += 1
	}
}

func generateBodyRequest(uri string, headers map[string]string, outputFile *fileStruct) {

	log.Printf("Testing: %s", uri)

	tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns: 1000,
		MaxIdleConnsPerHost: 1000,
    	}

	var netClient = &http.Client {
		Transport: tr,
		Timeout: time.Second * 10,
	}

	//Making string supplied in requestBody into a byte array to parse into request
	reqBody := []byte(requestBody)

	req, err := http.NewRequest(requestMethod, uri, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatal("Error making request: ", err)
	}

	for name, value := range headers {
		
		req.Header.Set(name,value)
	}

	res, err := netClient.Do(req)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		performGrep(uri, res, outputFile)
	}
}

func main() {

	flag.Parse()

	checkArguments()

	outputFile := newFile()

	requestsFileData = scanYamlInputFile(requestsFile)

	//Populate custom request headers, cookies, ...
	for _, v := range requestsFileData {
		parseRequestYaml(v)
	}

	//Get list of subdomains
	subdomains = scanTextInputFile(subdomainsFile)

	//Performing checks on the Request method supplied to ensure it is actually valid. If it is not, set it to GET as default
	requestMethod = strings.ToUpper(requestMethod)

	if !(requestMethod == "CONNECT" || requestMethod == "DELETE" || requestMethod == "GET" || requestMethod == "HEAD" || requestMethod == "OPTIONS" || requestMethod == "PATCH" || requestMethod == "POST" || requestMethod == "PUT" || requestMethod == "TRACE") {
		requestMethod = "GET"
	}

	wg := waitgroup.NewWaitGroup(50)

	//If a request body has been supplied
	if len(requestBody) > 0 {
		for _, subdomain := range subdomains {
			wg.BlockAdd()

			go func(subdomain string, customHeaders map[string]string, outputFile *fileStruct) {
				defer wg.Done()
				generateBodyRequest("http://" + subdomain + requestUrl, customHeaders, outputFile)
			}(subdomain, customHeaders, outputFile)
		}
	} else {
		for _, subdomain := range subdomains {
			wg.BlockAdd()

			go func(subdomain string, customHeaders map[string]string, outputFile *fileStruct) {
				defer wg.Done()
				generateRequest("http://" + subdomain + requestUrl, customHeaders, outputFile)
			}(subdomain, customHeaders, outputFile)
		}
	}

	wg.Wait()

	fmt.Println("Finished testing requests")
}
