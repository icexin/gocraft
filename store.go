package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

var (
	dbpath = flag.String("db", "gocraft.db", "db file name")
)

var (
	blockBucket  = []byte("block")
	chunkBucket  = []byte("chunk")
	cameraBucket = []byte("camera")

	store *Store
)

func InitStore() error {
	var path string
	if *dbpath != "" {
		path = *dbpath
	}
	if *serverAddr != "" {
		path = fmt.Sprintf("cache_%s.db", *serverAddr)
	}
	if path == "" {
		return errors.New("empty db path")
	}
	var err error
	store, err = NewStore(path)
	return err
}

type Store struct {
	db    *bolt.DB
	queue *WriteQueue
}

func NewStore(p string) (*Store, error) {
	db, err := bolt.Open(p, 0666, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(blockBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(chunkBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(cameraBucket)
		return err
	})
	if err != nil {
		return nil, err
	}
	db.NoSync = true
	s := &Store{
		db: db,
	}
	s.queue = NewWriteQueue(s, 500*time.Millisecond)
	return s, nil
}

func (s *Store) UpdateBlock(id Vec3, w int) {
	s.queue.Enqueue(id, w)
}

func (s *Store) updateBlockDirect(id Vec3, w int) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blockBucket)
		cid := id.Chunkid()
		key := encodeBlockDbKey(cid, id)
		value := encodeBlockDbValue(w)
		return bkt.Put(key, value)
	})
}

func (s *Store) UpdatePlayerState(state PlayerState) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(cameraBucket)
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, &state)
		bkt.Put(cameraBucket, buf.Bytes())
		return nil
	})
}

func (s *Store) GetPlayerState() PlayerState {
	var state PlayerState
	state.X = 5
	state.Y = float32(castleBaseY + 5)
	state.Z = 20
	s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(cameraBucket)
		value := bkt.Get(cameraBucket)
		if value == nil {
			return nil
		}
		buf := bytes.NewBuffer(value)
		binary.Read(buf, binary.LittleEndian, &state)
		return nil
	})
	return state
}

func (s *Store) RangeBlocks(id Vec3, f func(bid Vec3, w int)) error {
	return s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blockBucket)
		startkey := encodeBlockDbKey(id, Vec3{0, 0, 0})
		iter := bkt.Cursor()
		for k, v := iter.Seek(startkey); k != nil; k, v = iter.Next() {
			cid, bid := decodeBlockDbKey(k)
			if cid != id {
				break
			}
			w := decodeBlockDbValue(v)
			f(bid, w)
		}
		return nil
	})
}

func (s *Store) UpdateChunkVersion(id Vec3, version string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(chunkBucket)
		key := encodeVec3(id)
		return bkt.Put(key, []byte(version))
	})
}

func (s *Store) GetChunkVersion(id Vec3) string {
	var version string
	s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(chunkBucket)
		key := encodeVec3(id)
		v := bkt.Get(key)
		if v != nil {
			version = string(v)
		}
		return nil
	})
	return version
}

func (s *Store) Close() {
	s.queue.Stop()
	s.db.Sync()
	s.db.Close()
}

type writeEntry struct {
	id Vec3
	tp int
}

type WriteQueue struct {
	mu      sync.Mutex
	pending []writeEntry
	store   *Store
	ticker  *time.Ticker
	stopCh  chan struct{}
	done    chan struct{}
}

func NewWriteQueue(s *Store, interval time.Duration) *WriteQueue {
	q := &WriteQueue{
		store:  s,
		ticker: time.NewTicker(interval),
		stopCh: make(chan struct{}),
		done:   make(chan struct{}),
	}
	go q.run()
	return q
}

func (q *WriteQueue) Enqueue(id Vec3, tp int) {
	q.mu.Lock()
	q.pending = append(q.pending, writeEntry{id, tp})
	q.mu.Unlock()
}

func (q *WriteQueue) Flush() int {
	q.mu.Lock()
	entries := q.pending
	q.pending = nil
	q.mu.Unlock()
	if len(entries) == 0 {
		return 0
	}
	err := q.store.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(blockBucket)
		for _, e := range entries {
			cid := e.id.Chunkid()
			key := encodeBlockDbKey(cid, e.id)
			value := encodeBlockDbValue(e.tp)
			if err := bkt.Put(key, value); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("write queue flush error: %s", err)
		return 0
	}
	q.store.db.Sync()
	return len(entries)
}

func (q *WriteQueue) run() {
	defer close(q.done)
	for {
		select {
		case <-q.ticker.C:
			q.Flush()
		case <-q.stopCh:
			q.Flush()
			return
		}
	}
}

func (q *WriteQueue) Stop() {
	q.ticker.Stop()
	close(q.stopCh)
	<-q.done
}

func encodeVec3(v Vec3) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, [...]int32{int32(v.X), int32(v.Y), int32(v.Z)})
	return buf.Bytes()
}

func encodeBlockDbKey(cid, bid Vec3) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, [...]int32{int32(cid.X), int32(cid.Z)})
	binary.Write(buf, binary.LittleEndian, [...]int32{int32(bid.X), int32(bid.Y), int32(bid.Z)})
	return buf.Bytes()
}

func decodeBlockDbKey(b []byte) (Vec3, Vec3) {
	if len(b) != 4*5 {
		log.Panicf("bad db key length:%d", len(b))
	}
	buf := bytes.NewBuffer(b)
	var arr [5]int32
	binary.Read(buf, binary.LittleEndian, &arr)

	cid := Vec3{int(arr[0]), 0, int(arr[1])}
	bid := Vec3{int(arr[2]), int(arr[3]), int(arr[4])}
	if bid.Chunkid() != cid {
		log.Panicf("bad db key: cid:%v, bid:%v", cid, bid)
	}
	return cid, bid
}

func encodeBlockDbValue(w int) []byte {
	value := make([]byte, 4)
	binary.LittleEndian.PutUint32(value, uint32(w))
	return value
}

func decodeBlockDbValue(b []byte) int {
	if len(b) != 4 {
		log.Panicf("bad db value length:%d", len(b))
	}
	return int(binary.LittleEndian.Uint32(b))
}
