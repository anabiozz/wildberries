package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
)

func init() {
	log.SetFlags(0)
}

var searchWord = "Go"
var wordRegExp, _ = regexp.Compile("\\b" + searchWord + "\\b")

type wordCounter struct {
	wordCnts map[string]int
	total    int
	mux      sync.Mutex
}

func (c *wordCounter) count(url string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	words := wordRegExp.FindAllString(string(body), -1)

	c.mux.Lock()
	c.wordCnts = make(map[string]int)
	for _, word := range words {
		c.wordCnts[word]++
	}
	output := fmt.Sprintf("Count for %s: %d", url, c.wordCnts[searchWord])
	c.total += c.wordCnts[searchWord]
	c.mux.Unlock()

	log.Println(output)
}

func main() {
	var wg sync.WaitGroup

	maxGoroutines := 5
	scanner := bufio.NewScanner(os.Stdin)
	urlArray := []string{}

	// Reading url from stdin
	for scanner.Scan() {
		url := scanner.Text()
		urlArray = append(urlArray, url)
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln(err)
	}

	// Buff channel for coordination number of concurrent goroutins
	concurrentGoroutines := make(chan bool, maxGoroutines)
	// Fill this channel
	for i := 0; i < maxGoroutines; i++ {
		concurrentGoroutines <- true
	}

	// Channel for indication that single goroutine has finished its job
	goroutineIsDone := make(chan bool)
	counter := &wordCounter{}

	go func() {
		for i := 0; i < len(urlArray); i++ {
			// Check that one goroutine is finished its job
			<-goroutineIsDone

			// Wait until the channel is full or add element
			// to the channel to start new goroutine
			concurrentGoroutines <- true
		}
	}()

	for _, url := range urlArray {
		wg.Add(1)

		// When we have something, it means we can start a new
		// goroutine because another one finished
		<-concurrentGoroutines

		go func(url string) {
			// Start the job
			counter.count(url)
			// Send message that goroutine is finished its job
			goroutineIsDone <- true
			wg.Done()
		}(url)
	}

	wg.Wait()
	log.Printf("Total: %v", counter.total)
}
