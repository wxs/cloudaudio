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
		s := store.NewSession()
		var m = cloudaudio.SessionInfo{}
		m.IP = "127.0.0.1"
		m.Port = *dataport
		m.Sessid = s.Id
		m.AudioInfo = s.AudioInfo
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
	http.Handle("/audio/", http.StripPrefix("/audio/", http.HandlerFunc(audioHandler)))
}

func audioHandler(response http.ResponseWriter, request *http.Request) {
	path := request.URL.Path
	var id uint64
	fmt.Sscanf(path, "%d", &id)
	_, ok := store.GetSession(id)
	if !ok {
		http.Error(response,"That stream does not exist",404)
	}
}

func listenUDP(dataport int) {
	conn, err := net.ListenPacket("udp", fmt.Sprintf(":%d", dataport))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening for data on port:", dataport)
	for {
		b := make([]byte, 2048)
		n, addr, err := conn.ReadFrom(b)
		if err != nil {
			log.Fatal(err)
		}
		go func(b []byte, n int, addr net.Addr) {
			log.Printf("Just saw a packet! %d bytes, address %v\n", n, addr)
			log.Printf("%X%X%X%X%X%X\n", b[0], b[1], b[2], b[3], b[4], b[5])
			packet, err := cloudaudio.ParsePacket(b, n)
			if err != nil {
				log.Println("Warning: malformed packet", err)
				return
			}
			log.Println(packet)
			sess, ok := store.GetSession(packet.Id)
			if !ok {
				log.Println("Received packet for nonexistent session")
				return
			}
			sess.Packets <- packet
		}(b, n, addr)
	}
}
