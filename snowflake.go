/*
github.com/twitter/snowflake in golang
*/

package snowflake

import (
	"fmt"
	"hash/crc32"
	"math/rand"
	"net"
	"sync"
	"time"
)

const (
	WorkerIdBits = 10              // worker id
	MaxWorkerId  = -1 ^ (-1 << 10) // worker id mask
	SequenceBits = 12              // sequence
	MaxSequence  = -1 ^ (-1 << 12) // sequence mask
)

var (
	Since int64 = time.Date(2012, 1, 0, 0, 0, 0, 0, time.UTC).UnixNano() / int64(time.Millisecond)
)

type Snowflake struct {
	lastTimestamp uint64
	workerId      uint32
	sequence      uint32
	lock          sync.Mutex
}

func (sf *Snowflake) uint64() uint64 {
	return (sf.lastTimestamp << (WorkerIdBits + SequenceBits)) |
		(uint64(sf.workerId) << SequenceBits) |
		(uint64(sf.sequence))
}

func (sf *Snowflake) Next() (uint64, error) {
	sf.lock.Lock()
	defer sf.lock.Unlock()

	ts := timestamp()

	if ts == sf.lastTimestamp {
		sf.sequence = (sf.sequence + 1) & MaxSequence

		if sf.sequence == 0 {
			ts = tilNextMillis(ts)
		}
	} else {
		sf.sequence = 0
	}

	if ts < sf.lastTimestamp {
		return 0, fmt.Errorf("Invalid timestamp: %v - precedes %v", ts, sf)
	}

	sf.lastTimestamp = ts
	return sf.uint64(), nil
}

func Default() (*Snowflake, error) {
	return NewSnowflake(DefaultWorkId())
}

func NewSnowflake(workerId uint32) (*Snowflake, error) {
	if workerId < 0 || workerId > MaxWorkerId {
		return nil, fmt.Errorf("Worker id %v is invalid", workerId)
	}
	return &Snowflake{workerId: workerId}, nil
}

func timestamp() uint64 {
	return uint64(time.Now().UnixNano()/int64(time.Millisecond) - Since)
}

func tilNextMillis(ts uint64) uint64 {
	i := timestamp()

	for i <= ts {
		i = timestamp()
	}

	return i
}

func DefaultWorkId() uint32 {
	var id uint32
	ift, err := net.Interfaces()

	if err != nil {
		rand.Seed(time.Now().UnixNano())
		id = rand.Uint32() % MaxWorkerId
	} else {
		h := crc32.NewIEEE()

		for _, value := range ift {
			h.Write(value.HardwareAddr)
		}

		id = h.Sum32() % MaxWorkerId
	}

	return id & MaxWorkerId
}
