package cloudaudio

import (
	"container/heap"
	"crypto/rand"
	"encoding/binary"
	"io"
	"log"
	"runtime"
	"sync"
	"time"
)

type SessId uint32

var globalI int

type AudioInfo struct {
	SampleRate     int
	BytesPerSample int
	Channels       int
}

type Session struct {
	Id        SessId
	AudioInfo AudioInfo
	Packets   chan Packet

	quit          chan bool
	listeners     []chan Packet
	listenerMutex sync.RWMutex
}

// This type is used to communicate the important
// information about a session in the signalling server
type SessionInfo struct {
	IP        string
	Port      int
	Sessid    SessId
	AudioInfo AudioInfo
}

type SessionStore interface {
	GetSession(id SessId) *Session
	NewSession() *Session
	Sessions() []*Session
	Groups() SessionGroup
}

type SessionGroup struct {
	Sessions []*Session
}

type DefaultSessionStore struct {
	mutex    sync.RWMutex
	sessions map[SessId]*Session
}

func (s *DefaultSessionStore) GetSession(id SessId) (session *Session, ok bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	session, ok = s.sessions[id]
	return
}

type sessionReader struct {
	// How far have we read thus far? We can't
	// insert new packets early than this or
	// we'll introduce jitter
	offset   int64
	quit     chan bool
	pq       PacketQueue
	mutex    sync.Mutex
	currPack *Packet
	capacity int
}

// This gives an object implementing io.ReadCloser that
// first buffers at least buffLength packets and reorders
// them to be properly sorted as some degree of protection
// against jitter 'n' such.
func (s *Session) Reader(buffLength int) io.ReadCloser {
	packets := make(chan Packet, buffLength)
	s.AddListener(packets)
	r := new(sessionReader)
	r.quit = make(chan bool)
	r.pq = make(PacketQueue, 0, buffLength+buffLength/2)
	r.capacity = buffLength + buffLength/2
	heap.Init(&r.pq)
	go func(packets chan Packet, r *sessionReader, id int) {
		log.Println("Building reader", id)
		for {
			select {
			case pack := <-packets:
				r.mutex.Lock()
				for r.pq.Len() >= r.capacity {
					heap.Pop(&r.pq)
				}
				if pack.Count >= r.offset {
					heap.Push(&r.pq, &pack)
				}
				r.mutex.Unlock()
			case <-r.quit:
				log.Println("Quitting")
				return
			}
		}
	}(packets, r, globalI)
	globalI++
	log.Println("Reader created")
	return r
}

func (r *sessionReader) Read(p []byte) (n int, err error) {
	r.mutex.Lock()
	defer runtime.Gosched() // Otherwise this makes copies block forever
	defer r.mutex.Unlock()  // Unlock before scheduling others; we're about to return anyway
	n = 0
	if r.currPack == nil {
		// If we're not yet reading from any packet, grab
		// one from the queue
		if r.pq.Len() == 0 {
			return
		}
		r.currPack = heap.Pop(&r.pq).(*Packet)
	}
	p2 := p
	for n < len(p) {
		// First keep reading off the current packet
		start := r.currPack.Count
		packOffset := int32(r.offset - start)
		if packOffset < 0 {
			r.offset = r.currPack.Count
			packOffset = 0
		}
		if packOffset < r.currPack.Size {
			copied := copy(p2, r.currPack.Payload[packOffset:])
			n += copied
			r.offset += int64(copied)
			if n == len(p) {
				return
			}
			p2 = p2[copied:]
		}
		// Ok, we've exhausted that packet. Next!
		if r.pq.Len() == 0 {
			return
		}
		r.currPack = heap.Pop(&r.pq).(*Packet)
	}
	return n, Error("sessionRead terminated in an unexpected state")
}
func (r sessionReader) Close() error {
	go func() { r.quit <- true }()
	return nil
}

func (s *Session) AddListener(c chan Packet) {
	s.listenerMutex.Lock()
	defer s.listenerMutex.Unlock()
	s.listeners = append(s.listeners, c)
	log.Println("Added a listener, new number: ", len(s.listeners))
}

func sessionHandler(s *Session) {
	for {
		// Loop forever waiting for new parsed packets to show up
		select {
		case pack := <-s.Packets:
			s.listenerMutex.RLock()
			for _, listener := range s.listeners {
				go func(l chan Packet) {
					timeout := make(chan bool, 1)
					go func() {
						time.Sleep(1 * time.Second)
						timeout <- true
					}()
					select {
					case l <- pack:
					case <-timeout:
					}
				}(listener)
			}
			s.listenerMutex.RUnlock()
		case <-s.quit:
			return

		}
	}
}

func (s *DefaultSessionStore) NewSession() Session {
	id := genSessId()
	_, ok := s.GetSession(id)
	for ok {
		id = genSessId()
		_, ok = s.GetSession(id)
	}
	info := AudioInfo{44100, 16, 1}
	var r Session
	r.Id = id
	r.AudioInfo = info
	r.Packets = make(chan Packet)
	r.quit = make(chan bool)
	r.listeners = make([]chan Packet, 0, 10)
	go sessionHandler(&r)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sessions[id] = &r
	return r
}

func (s *DefaultSessionStore) Sessions() []*Session {

	r := make([]*Session, len(s.sessions))
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	i := 0
	for _, sess := range s.sessions {
		r[i] = sess
		i++
	}
	return r
}

func NewDefaultSessionStore() *DefaultSessionStore {
	r := new(DefaultSessionStore)
	r.sessions = make(map[SessId]*Session)
	return r
}

func genSessId() SessId {
	var r SessId
	err := binary.Read(rand.Reader, binary.LittleEndian, &r)
	if err != nil {
		log.Fatal("genSessID:", err)
	}
	return r
}
