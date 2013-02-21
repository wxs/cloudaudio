package cloudaudio

import (
	"crypto/rand"
	"encoding/binary"
	"log"
	"sync"
)

type SessionStore interface {
	GetSession(id uint64) Session
	NewSession() Session
	Sessions() []Session
}

type DefaultSessionStore struct {
	mutex    sync.RWMutex
	sessions map[uint64]Session
}

func (s *DefaultSessionStore) GetSession(id uint64) (session Session, ok bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	session, ok = s.sessions[id]
	return
}

func (s *DefaultSessionStore) NewSession() Session {
	id := genSessId()
	_, ok := s.GetSession(id)
	for ok {
		id = genSessId()
		_, ok = s.GetSession(id)
	}
	info := AudioInfo{44100, 16, 1}
	r := Session{id, info}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sessions[id] = r

	return r
}

func (s *DefaultSessionStore) Sessions() []Session {

	r := make([]Session, len(s.sessions))
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
	r.sessions = make(map[uint64]Session)
	return r
}

func genSessId() uint64 {
	var r uint64
	err := binary.Read(rand.Reader, binary.LittleEndian, &r)
	if err != nil {
		log.Fatal("genSessID:", err)
	}
	return r
}

type AudioInfo struct {
	SampleRate     int
	BytesPerSample int
	Channels       int
}

type Session struct {
	Id        uint64
	AudioInfo AudioInfo
}

// This type is used to communicate the important
// information about a session in the signalling server
type SessionInfo struct {
	IP        string
	Port      int
	Sessid    uint64
	AudioInfo AudioInfo
}
