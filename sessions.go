package cloudaudio

import (
    "fmt"
    "log"
    "io"
    "crypto/rand"
)

func genSessId() uint64 {
    b := make([]byte, 8);
    n, err:=io.ReadFull(rand.Reader,b)
    if n!=len(b) || err!=nil {
        log.Fatal("genSessID:",err)
    }
    var r uint64 = 0
    for i:=0;i<len(b);i++ {
        r=r<<8;
        r|=uint64(b[i])
    }
    return r
}

type Session struct {
    id uint64
}

func NewSession() Session {
    r := Session{genSessId()}
    return r
}

func (s Session) HexId() string {
    return fmt.Sprintf("%016x",s.id)
}

