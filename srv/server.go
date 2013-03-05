package main

import (
	"cloudaudio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
)

const (
	staticPath = "static/"
)

var dataport = flag.Int("pd", 2445, "The port to listen for UDP data packets")
var signalport = flag.Int("ps", 2444, "The port to listen for bidirectional signalling")
var echoPackets = flag.Bool("vp", false, "Should we echo packets to stdout?")

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
		ids := make([]cloudaudio.SessId, len(sessions))
		for i, s := range sessions {
			ids[i] = s.Id
		}
		b, err := json.Marshal(ids)
		if err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
		}
		response.Header().Add("Content-type", "application/json")
		response.Write(b)
	})
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir(staticPath))))
	http.Handle("/audio/", http.StripPrefix("/audio/", http.HandlerFunc(audioHandler)))
}

func audioHandler(response http.ResponseWriter, request *http.Request) {
	path := request.URL.Path
	var id cloudaudio.SessId
	fmt.Sscanf(path, "%d", &id)
	sess, ok := store.GetSession(id)
	log.Println(sess)
	if !ok {
		http.Error(response, fmt.Sprintf("That stream: %d does not exist", id), 404)
		return
	}
	packets := make(chan cloudaudio.Packet, 20)
	sess.AddListener(packets)

	response.Header().Add("Content-type", "audio/mpeg")

	// spawn a process transcoding the stream
	// In reality, of course, having multiple processes serving up the
	// same stream to each client is stupid.
	//cmd := exec.Command("vlc",
	//"--demux=rawaud", "--rawaud-channels=1", "--rawaud-samplerate=44100", "--rawaud-fourcc=s16l",
	//"-", "--sout" , "'#transcode{acodec='mpga'}#standard{access=http,mux='ts'}
	cmd := exec.Command("ffmpeg", "-v", "verbose", "-f", "u16le", "-ar", "44100", "-ac", "1", "-i", "-",
		"-f", "mp3", "-")
	//cmd := exec.Command("oggenc", "-r", "-C", "1", "-")

	audioIn, err := cmd.StdinPipe()
	if err != nil {
		log.Println("Failed to create stdin pipe")
	}
	audioOut, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Failed to create stdout pipe")
	}
	audioErr, err := cmd.StderrPipe()
	if err != nil {
		log.Println("Failed to create stderr pipe")
	}
	go io.Copy(os.Stderr, audioErr)
	err = cmd.Start()
	if err != nil {
		log.Println("Failed to start ffmpeg command, error: ", err)
	}
	log.Println("Running the command")
	go func() {
		for {
			packet := <-packets
			audioIn.Write(packet.Payload)
			if err != nil {
				log.Println("Error in the audio streaming handler")
				return
			}
		}
	}()
	go func() {
		amount, err := io.Copy(response, audioOut)
		if err != nil {
			log.Println("io copy terminated with an error", err)
		}
		log.Printf("Done copying audio data: %d bytes\n", amount)
	}()
	err = cmd.Wait()
	if err != nil {
		log.Println("ffmpeg command terminated incorrectly", err)
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
			if *echoPackets {
				log.Printf("Just saw a packet! %d bytes, address %v\n", n, addr)
				log.Printf("%X%X%X%X%X%X\n", b[0], b[1], b[2], b[3], b[4], b[5])
			}
			packet, err := cloudaudio.ParsePacket(b, n)
			if err != nil {
				log.Println("Warning: malformed packet", err)
				return
			}
			if *echoPackets {
				log.Println(packet)
			}
			sess, ok := store.GetSession(packet.Id)
			if !ok {
				log.Println("Received packet for nonexistent session")
				return
			}
			sess.Packets <- packet
		}(b, n, addr)
	}
}
