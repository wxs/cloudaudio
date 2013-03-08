package main

import (
	"cloudaudio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

var nstreams = flag.Int("n", 1, "The number of concurrent streams")
var signalserver = flag.String("srv", "http://localhost:2444", "The server to connect to for signalling")
var slow = flag.Bool("s", false, "Run slowly?")

func main() {
	flag.Parse()
	log.Printf("Starting up %d mock streams\n", *nstreams)
	quit := make(chan int)
	for i := 0; i < *nstreams; i++ {
		go func(id int) {
			log.Println("Starting stream ", id)
			stream(id)
			quit <- 1
		}(i)
	}
	<-quit
	log.Println("Abnormal mock stream termination")
}

func stream(streamid int) {
	fname := fmt.Sprintf("%d.raw", streamid)
	file, err := os.Open(fname)
	if err != nil {
		log.Fatal("Could not open file", fname)
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal("Error opening the file")
	}
	connecturl := fmt.Sprintf("%s/connect", *signalserver)
	resp, err := http.Get(connecturl)
	if err != nil {
		log.Fatal("Could not connect to signalling server: ", err)
	}
	dec := json.NewDecoder(resp.Body)
	var s cloudaudio.SessionInfo
	dec.Decode(&s)
	resp.Body.Close()
	log.Println("Started stream with id: ", s.Sessid)
	var count int64 = 0
	// Buffer half a second of fake audio at a time
	buff := make([]byte, 1024)

	addr := net.UDPAddr{net.ParseIP(s.IP), s.Port}
	conn, err := net.DialUDP("udp", nil, &addr)
	if err != nil {
		log.Fatal("Could not connect to UDP server:", err)
	}
	buffNanos := 1e9 / s.AudioInfo.SampleRate / s.AudioInfo.BytesPerSample * len(buff)
	var tick <-chan time.Time
	if *slow {
		tick = time.Tick(time.Second)
	} else {
		tick = time.Tick(time.Nanosecond * time.Duration(buffNanos))
	}
	byteOff := 0
	for now := range tick {
		nanos := now.UnixNano()
		for i := range buff {
			buff[i] = bytes[byteOff]
			byteOff = (byteOff + 1) % len(bytes)
		}
		packet := cloudaudio.Packet{s.Sessid, nanos, count, int32(len(buff)), buff}
		b, err := packet.Bytes()
		if err != nil {
			log.Fatal("Failed to create audio packet: ", err)
		}
		count += int64(len(buff))
		n, err := conn.Write(b)
		if err != nil {
			log.Fatal("Failed to send UDP packet: ", err)
		}
		if n != len(b) {
			log.Fatal("Failed to send full payload of UDP packet")
		}
	}
}
