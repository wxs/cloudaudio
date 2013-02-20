package cloudaudio

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

type Packet struct {
	Id      uint64
	Time    uint64
	Count   uint32
	Payload []byte
}

func ParsePacket(b []byte, n int) (p Packet, err error) {
	buf := bytes.NewBuffer(b)

	e := binary.Read(buf, binary.LittleEndian, &p.Id)
	if e != nil {
		err = Error("Failed to read id")
		return
	}
	e = binary.Read(buf, binary.LittleEndian, &p.Time)
	if e != nil {
		err = Error("Failed to read time")
		return
	}
	e = binary.Read(buf, binary.LittleEndian, &p.Count)
	if e != nil {
		err = Error("Failed to read count")
		return
	}
	// This may be an unecessary copy; there could be a way to do this
	// using the same underlying array and slices...
	p.Payload, e = ioutil.ReadAll(buf)
	if e != nil {
		err = Error("Failed to read packet payload")
		return
	}
	return
}
