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
    log.Println("Called publish")
    endpoint := fmt.Sprintf("%s/pub", p.addr)
    var buffer bytes.Buffer
    buffer.Write([]byte(msg))
    resp, err := http.Post(endpoint, "application/json", &buffer)
    defer resp.Body.Close()
    return err
}

func PublishLoop(pub Publisher, publishMsgs chan string) {
    for {
        msg := <-publishMsgs
        log.Println("Publishing message: ", msg)

        err := pub.Publish(msg)
        if err != nil {
            log.Println("ERROR publishing: ", err)
            break
        }
    }
}


func main() {
    flag.Parse()
    var publisher Publisher
    if len(*pubsub) > 0 {
        log.Println("Pubsub")
        publisher = &PubsubPublisher{PublisherInfo{&http.Client{}, *pubsub}}
    }

    msgsChan := make(chan string, 1) // TODO - decide how much we wish to buffer here
    go PublishLoop(publisher, msgsChan)
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
}
