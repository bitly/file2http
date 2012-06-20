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
    "bytes"
    "log"
)

var pubsub = flag.String("pubsub", "", "Pubsub to write to")

type Publisher interface {
    Publish(string) error
}

type PublisherInfo struct { // ha! this is what happens when your language makes you use interfaces
    httpclient *http.Client
    addr string
}

type PubsubPublisher struct {
    PublisherInfo
}

func (p *PubsubPublisher) Publish(msg string) error {
    endpoint := fmt.Sprintf("%s/pub", p.addr)
    var buffer bytes.Buffer
    buffer.Write([]byte(msg))
    resp, err := http.Post(endpoint, "application/json", &buffer)
    defer resp.Body.Close()
    return err
}

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
    var publisher Publisher
    if len(*pubsub) > 0 {
        publisher = &PubsubPublisher{PublisherInfo{&http.Client{}, *pubsub}}
    }

    msgsChan := make(chan string, 1) // TODO - decide how much we wish to buffer here
    publishExitChan := make(chan bool)
    go PublishLoop(publisher, msgsChan, publishExitChan)
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
    publishExitChan <- true
}
