package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime/pprof"
	"strings"
)

var post = flag.String("post", "", "Address to make a POST request to. Data will be in the body")
var get = flag.String("get", "", `Address to make a GET request to.
     Address should be a format string where data can be subbed in`)
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var numPublishers = flag.Int("n", 5, "Number of concurrent publishers")
var showVersion = flag.Bool("version", false, "print version string")

const VERSION = "0.1"

type Publisher interface {
	Publish(string) error
}

type PublisherInfo struct {
	addr string
}

type PostPublisher struct {
	PublisherInfo
}

func (p *PostPublisher) Publish(msg string) error {
	reader := bytes.NewReader([]byte(msg))
	resp, err := http.Post(p.addr, "application/octet-stream", reader)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return err
}

type GetPublisher struct {
	PublisherInfo
}

func (p *GetPublisher) Publish(msg string) error {
	endpoint := fmt.Sprintf(p.addr, url.QueryEscape(msg))
	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return err
}

func PublishLoop(done chan struct{}, pub Publisher, publishMsgs chan string) {
	for msg := range publishMsgs {
		err := pub.Publish(msg)
		if err != nil {
			log.Println("ERROR publishing: ", err)
			break
		}
	}
	done <- struct{}{}
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("file2http v%s\n", VERSION)
		return
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	var publisher Publisher
	if len(*post) > 0 {
		publisher = &PostPublisher{PublisherInfo{*post}}
	} else if len(*get) > 0 {
		if strings.Count(*get, "%s") != 1 {
			log.Fatal("Invalid get address - must be a format string")
		}
		publisher = &GetPublisher{PublisherInfo{*get}}
	} else {
		log.Fatal("Need get or post address!")
	}

	msgsChan := make(chan string)
	publishExitChan := make(chan struct{})
	for i := 0; i < *numPublishers; i++ {
		go PublishLoop(publishExitChan, publisher, msgsChan)
		// TODO - crazy idea: what if I were to defer reading from the exit channel here?
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Println(fmt.Sprintf("ERROR: %s", err))
			}
			break
		}
		line = strings.TrimSpace(line)
		msgsChan <- line
	}
	close(msgsChan)
	for i := 0; i < *numPublishers; i++ {
		<-publishExitChan
	}
	close(publishExitChan)
}
