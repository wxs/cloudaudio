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
	Time    int64 //Nanosecond timestamp
	Count   int32
	Payload []byte
}

func (p Packet) Bytes() ([]byte, error) {
	b := make([]byte, 8+8+4+len(p.Payload))
	buff := bytes.NewBuffer(b)
	if err := binary.Write(buff, binary.LittleEndian, p.Id); err != nil {
		return b, err
	}
	if err := binary.Write(buff, binary.LittleEndian, p.Time); err != nil {
		return b, err
	}
	if err := binary.Write(buff, binary.LittleEndian, p.Count); err != nil {
		return b, err
	}
	if err := binary.Write(buff, binary.LittleEndian, p.Payload); err != nil {
		return b, err
	}
	return b, nil
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
