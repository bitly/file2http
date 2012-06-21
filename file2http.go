package main

import (
    // "bitly/simplejson"
    "flag"
    "fmt"
    "bufio"
    "os"
    "strings"
    "io"
    "net/http"
    "net/url"
    "bytes"
    "log"
    "strconv"
    "runtime/pprof"
)

var pubsub = flag.String("p", "", "Pubsub address (or local port) to write to")
var simplequeue = flag.String("s", "", "Simplequeue address (or local port) to write to")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var numPublishers = flag.Int("n", 5, "Number of concurrent publishers")

func ParseAddress(inputted string) string {
    port, err := strconv.Atoi(inputted)
    if err == nil {
        // assume this was meant to be a port number
        return fmt.Sprintf("http://127.0.0.1:%d", port)
    }
    return inputted
}

type Publisher interface {
    Publish(string) error
}

type PublisherInfo struct { // ha! this is what happens when your language makes you use interfaces
    httpclient *http.Client
    addr string
}

// ---------- PubSub -------------------------

type PubsubPublisher struct {
    PublisherInfo
}

func (p *PubsubPublisher) Publish(msg string) error {
    endpoint := fmt.Sprintf("%s/pub", p.addr)
    var buffer bytes.Buffer
    buffer.Write([]byte(msg))
    resp, err := http.Post(endpoint, "application/json", &buffer)
    defer resp.Body.Close()
    defer buffer.Reset()
    return err
}


// ----------- SQ ---------------------------

type SimplequeuePublisher struct {
    PublisherInfo
}

func (p *SimplequeuePublisher) Publish(msg string) error {
    endpoint := fmt.Sprintf("%s/put?data=%s", p.addr, url.QueryEscape(msg))
    resp, err := http.Get(endpoint)
    defer resp.Body.Close()
    return err
}

// ---------- Main Logic ----------------------

func PublishLoop(pub Publisher, publishMsgs chan string, exitChan chan bool) {
    exit := false
    for {
        var msg string
        select {
        case msg = <-publishMsgs:
        // log.Println("Publishing message: ", msg)

        err := pub.Publish(msg)
        if err != nil {
            log.Println("ERROR publishing: ", err)
            exit = true
        }
        case <- exitChan:
            exit = true
        }
        if exit {
            break
        }
    }
}


func main() {
    flag.Parse()
    if *cpuprofile != "" {
        f, err := os.Create(*cpuprofile)
        if err != nil {
            log.Fatal(err)
        }
        pprof.StartCPUProfile(f)
        defer pprof.StopCPUProfile()
    }
    var publisher Publisher
    if len(*pubsub) > 0 {
        publisher = &PubsubPublisher{PublisherInfo{&http.Client{}, ParseAddress(*pubsub)}}
    }else if len(*simplequeue) > 0 {
        publisher = &SimplequeuePublisher{PublisherInfo{&http.Client{}, ParseAddress(*simplequeue)}}
    }

    msgsChan := make(chan string, 1) // TODO - decide how much we wish to buffer here
    publishExitChan := make(chan bool)
    for i := 0; i < *numPublishers; i++ {
        go PublishLoop(publisher, msgsChan, publishExitChan)
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
    for i := 0; i < *numPublishers; i++ {
        publishExitChan <- true
    }
}
