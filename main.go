package main

import (
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	client = &http.Client{Transport: &http.Transport{
		MaxIdleConns:        2500,
		MaxIdleConnsPerHost: 2500,
		IdleConnTimeout:     60 * time.Second,
	}}
	headers = map[string]string{
		"User-Agent": "load-generator",
	}
	reqCounter uint64
	logFile    = "/tmp/lclient.log"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

var url string = "http://"

func generateRandomString(len_ int) string {
	str := make([]byte, len_)
	for i := range str {
		str[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(str)
}

func generateHeaders(count int) {
	for i := 0; i < count; i++ {
		header, value := generateRandomString(10), generateRandomString(12)
		headers[header] = value
	}
}

func sendRequest() {
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
	}
	atomic.AddUint64(&reqCounter, 1)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU()) // UTILIZE all cpu cores
	// Create service log file
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Can't create or open file: ", logFile)
	}
	defer f.Close()
	log.SetOutput(f)
	log.Println("Start logging to file! Service: load-generator-client")

	// Log number of utilized cpus
	log.Println("NumCPU is ", runtime.NumCPU())

	// Set up server host url
	log.Println("Set up server url...")
	server_hostname, exists := os.LookupEnv("SERVER_HOSTNAME")
	if !exists {
		log.Fatal("Server hostname is not set!")
	}
	server_port, exists := os.LookupEnv("SERVER_PORT")
	if !exists {
		log.Fatal("Server port is not set!")
	}
	url += server_hostname + ":" + server_port + "/get_random_json_1"
	log.Println("Done!")

	// Check if the Host is reachable...
	log.Println("Check if the server is reachable...")
	if _, err := net.DialTimeout("tcp", "localhost:6080", 1*time.Second); err != nil {
		log.Fatal("Host is unreachable: ", err)
	}
	log.Println("Done!")

	// Generate Headers
	log.Println("Generate headers...")
	generateHeaders(50)
	log.Println("Done!")

	// Start sending requests...
	log.Println("Start sending requests...")
	for i := 0; i < runtime.NumCPU()*2; i++ {
		go func() {
			for {
				sendRequest()
			}
		}()
	}
	log.Println("Done")

	// Start monitoring requests
	log.Println("Start RPS monitoring...")
	go func() {
		for {
			time.Sleep(time.Second)
			rps := atomic.SwapUint64(&reqCounter, 0)
			log.Println("RPS:", rps)
		}
	}()

	select {}
}

