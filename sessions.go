package cloudaudio

import (
	"crypto/rand"
	"encoding/binary"
	"log"
	"sync"
	"time"
)

type SessId uint32

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
				go func() {
					timeout := make(chan bool, 1)
					go func() {
						time.Sleep(1 * time.Second)
						timeout <- true
					}()
					select {
					case listener <- pack:
					case <-timeout:
					}
				}()
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
