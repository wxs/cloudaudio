package cloudaudio

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

type Packet struct {
	Id      uint64
	Time    int64 //Nanosecond timestamp
	Count   int32
	Size    int32
	Payload []byte
}

func (p Packet) String() string {
	return fmt.Sprintf("Packet {Id: %d, Time: %d, Count: %d, Payload: %d bytes}",
		p.Id,
		p.Time,
		p.Count,
		p.Size)
}

func (p Packet) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	var metadata = []interface{}{p.Id, p.Time, p.Count, p.Size, p.Payload}
	for _, v := range metadata {
		if err := binary.Write(&buf, binary.LittleEndian, v); err != nil {
			return nil, err
		}
	}
	b := buf.Bytes()
	return b, nil
}

func ParsePacket(b []byte, n int) (p Packet, err error) {
	buf := bytes.NewBuffer(b)
	metadata := struct {
		Id    uint64
		Time  int64
		Count int32
		Size  int32
	}{}
	e := binary.Read(buf, binary.LittleEndian, &metadata)
	if e != nil {
		err = Error("Failed to read packet metadata")
		return
	}
	p.Id = metadata.Id
	p.Time = metadata.Time
	p.Count = metadata.Count
	p.Size = metadata.Size
	// This may be an unecessary copy; there could be a way to do this
	// using the same underlying array and slices...
	p.Payload = make([]byte, p.Size)
	e = binary.Read(buf, binary.LittleEndian, p.Payload)
	if e != nil {
		err = Error("Failed to read packet payload")
		return
	}
	if int32(len(p.Payload)) != p.Size {
		err = Error(
			fmt.Sprintf("Malformed packet: incorrect payload size; claimed %d bytes, saw %d bytes",
				p.Size, len(p.Payload)))
	}
	return
}
