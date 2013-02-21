package main

import (
	"cloudaudio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"
)

var nstreams = flag.Int("n", 1, "The number of concurrent streams")
var signalserver = flag.String("srv", "http://localhost:2444", "The server to connect to for signalling")

func main() {
	flag.Parse()
	for i := 0; i < *nstreams; i++ {
		go stream()
	}
	for {
	}

}

func stream() {
	log.Println("About to connect to signalling server")
	connecturl := fmt.Sprintf("http://localhost:%s/connect", *signalserver)
	resp, err := http.Get(connecturl)
	if err != nil {
		log.Fatal("Could not connect to signalling server:", err)
	}
	dec := json.NewDecoder(resp.Body)
	var s cloudaudio.SessionInfo
	dec.Decode(&s)
	resp.Body.Close()
	log.Println("Started stream with id: ", s.Sessid)
	var count int32 = 0
	// Buffer half a second of fake audio at a time
	buff := make([]byte, s.AudioInfo.SampleRate/2*s.AudioInfo.BytesPerSample)

	addr := net.UDPAddr{net.ParseIP(s.IP), s.Port}
	conn, err := net.DialUDP("udp", nil, &addr)
	if err != nil {
		log.Fatal("Could not connect to UDP server:", err)
	}
	for {
		start := time.Now()
		nanos := start.UnixNano()
		for i := range buff {
			buff[i] = byte(rand.Uint32() & 0xff)
		}
		packet := cloudaudio.Packet{s.Sessid, nanos, count, buff}
		b, err := packet.Bytes()
		if err != nil {
			log.Fatal("Failed to create audio packet")
		}
		count++
		n, err := conn.WriteToUDP(b, &addr)
		if err != nil {
			log.Fatal("Failed to send UDP packet")
		}
		if n != len(b) {
			log.Fatal("Failed to send full payload of UDP packet")
		}
		end := time.Now()
		halfsec, _ := time.ParseDuration("0.5s")
		nextStart := start.Add(halfsec)
		time.Sleep(nextStart.Sub(end))
	}
}
