// The PacketQueue is a priority queue of packets ordered by their
// sample count; it is useful for jitter reduction.
package cloudaudio

type PacketQueue []*Packet

func NewPacketQueue(capacity int) *PacketQueue {
	h := make(PacketQueue, 0, capacity)
	return &h
}

// Much of this code shamelessly stolen from http://golang.org/pkg/container/heap
func (pq PacketQueue) Len() int {
	return len(pq)
}

func (pq PacketQueue) Less(i, j int) bool {
	return pq[i].Count < pq[j].Count
}

func (pq PacketQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PacketQueue) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	a := append(*pq, x.(*Packet))
	*pq = a
}

func (pq *PacketQueue) Pop() interface{} {
	a := *pq
	n := len(a)
	packet := a[n-1]
	*pq = a[0 : n-1]
	return packet
}
