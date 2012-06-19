package main

import (
    // "bitly/simplejson"
    "flag"
    "fmt"
    "bufio"
    "os"
    "strings"
    "io"
)

var pubsub = flag.String("pubsub", "", "Pubsub to write to")




func main() {
    flag.Parse()
    reader := bufio.NewReader(os.Stdin)
    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            if err != io.EOF {
                fmt.Println(fmt.Sprintf("ERROR: %s", err))
            }
            break
        }
        line = strings.TrimSpace(line)
        fmt.Println(fmt.Sprintf(">> %s", line))
    }
}
