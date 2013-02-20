package main

import (
	"cloudaudio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
)

var dataport = flag.Int("pd", 2445, "The port to listen for UDP data packets")
var signalport = flag.Int("ps", 2444, "The port to listen for bidirectional signalling")

var store *cloudaudio.DefaultSessionStore
var sessionChannels map[uint64]chan cloudaudio.Packet

func main() {
	log.Println("Starting server")
	flag.Parse()

	store = cloudaudio.NewDefaultSessionStore()

	// Start a goroutine whose whole purpose is to listen to
	// the incoming UDP stream
	// and another for the signalling
	go listenUDP(*dataport)
	configureHTTP()
	log.Println("Listening for signalling on port:", *signalport)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *signalport), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func configureHTTP() {
	http.HandleFunc("/connect", func(response http.ResponseWriter, request *http.Request) {
		type SessionInfo struct {
			Host     string
			Port     int
			Sessid   string
			MaxBytes int
			audio    cloudaudio.AudioInfo
		}
		s := store.NewSession()
		var m = SessionInfo{}
		m.Host = "127.0.0.1"
		m.Port = *dataport
		m.Sessid = s.HexId()
		m.MaxBytes = 1024
		m.audio = s.AudioInfo
		b, err := json.Marshal(m)
		if err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
		}
		response.Header().Add("Content-type", "application/json")
		response.Write(b)
	})
	http.HandleFunc("/sessions", func(response http.ResponseWriter, request *http.Request) {
		response.Header().Add("Content-type", "application/json")
		sessions := store.Sessions()
		b, err := json.Marshal(sessions)
		if err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
		}
		response.Header().Add("Content-type", "application/json")
		response.Write(b)
	})

}

func listenUDP(dataport int) {
	conn, err := net.ListenPacket("udp", fmt.Sprintf(":%d", dataport))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening for data on port:", dataport)
	for {
		b := make([]byte, 1024)
		n, addr, err := conn.ReadFrom(b)
		if err != nil {
			log.Fatal(err)
		}
		go func(b []byte, n int, addr net.Addr) {
			log.Printf("Just saw a packet! %d bytes, address %v\n", n, addr)
			log.Printf("%s\n", b[0:n])
			packet, err := cloudaudio.ParsePacket(b, n)
			if err != nil {
				log.Println("Warning: malformed packet", err)
				return
			}
			ch := sessionChannels[packet.Id]
			if ch == nil {
				log.Println("Received packet for nonexistent session")
				return
			}
			ch <- packet
		}(b, n, addr)
	}
}
