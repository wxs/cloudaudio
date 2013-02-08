package main

import (
    "log"
    "net"
    "net/http"
    "fmt"
    "flag"
    "encoding/json"
    "cloudaudio"
)

var dataport = flag.Int("pd", 2445, "The port to listen for UDP data packets")
var signalport = flag.Int("ps", 2444, "The port to listen for bidirectional signalling")


func main() {
    log.Println("Starting server")

    quit := make(chan int)
    // Start a goroutine whose whole purpose is to listen to
    // the incoming UDP stream
    flag.Parse()
    go listenHTTP(*signalport, quit)
    go listenUDP(*dataport, quit)

    // Also start a signalling goroutine


    <-quit; // Block until the servers terminate
    log.Println("One of the servers terminated")
}


func listenHTTP(signalport int, quit chan int) {
        http.HandleFunc("/connect", func(response http.ResponseWriter, request *http.Request) {
            response.Header().Add("Content-type", "application/json")
            type SessionInfo struct {
                Host string
                Port int
                Sessid string
                MaxBytes int
            }
            s := cloudaudio.NewSession()
            var m = SessionInfo{}
            m.Host = "127.0.0.1"
            m.Port = *dataport
            m.Sessid = s.HexId()
            m.MaxBytes=1024
            b,err := json.Marshal(m)
            if err != nil {
                log.Fatal("listenHTTP: ", err);
            }
            response.Write(b)
        })
        log.Println("Listening for signalling on port:", signalport)
        err := http.ListenAndServe(fmt.Sprintf(":%d",signalport), nil);
        if err != nil {
            log.Fatal("ListenAndServe: ", err)
        }
}

func listenUDP(dataport int, quit chan int) {
        conn,err := net.ListenPacket("udp",fmt.Sprintf(":%d",dataport))
        if err != nil {
            log.Fatal(err)
        }
        log.Println("Listening for data on port:", dataport)
        for {
            b := make([]byte,1024)
            n, addr, err := conn.ReadFrom(b)
            if err != nil {
                log.Fatal(err)
                quit<-1
            }
            go func(b []byte, n int, addr net.Addr) {
                log.Printf("Just saw a packet! %d bytes, address %v\n", n,addr)
                log.Printf("%s\n", b[0:n])
            }(b,n,addr)
        }
}

