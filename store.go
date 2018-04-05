package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"log"

	"github.com/boltdb/bolt"
	"github.com/go-gl/mathgl/mgl32"
)

var (
	dbpath = flag.String("db", "gocraft.db", "db file name")
)

var (
	blockBucket  = []byte("block")
	cameraBucket = []byte("camera")

	store *Store
)

func InitStore() error {
	if *dbpath == "" {
		return nil
	}
	var err error
	store, err = NewStore(*dbpath)
	return err
}

type Store struct {
	db *bolt.DB
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
		_, err = tx.CreateBucketIfNotExists(cameraBucket)
		return err
	})
	if err != nil {
		return nil, err
	}
	db.NoSync = true
	return &Store{
		db: db,
	}, nil
}

func (s *Store) UpdateBlock(id Vec3, w int) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		log.Printf("put %v -> %d", id, w)
		bkt := tx.Bucket(blockBucket)
		cid := id.Chunkid()
		key := encodeBlockDbKey(cid, id)
		value := encodeBlockDbValue(w)
		return bkt.Put(key, value)
	})
}

func (s *Store) UpdateCamera(pos mgl32.Vec3, rx, ry float32) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(cameraBucket)
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, &pos)
		binary.Write(buf, binary.LittleEndian, [...]float32{rx, ry})
		bkt.Put(cameraBucket, buf.Bytes())
		return nil
	})
}

func (s *Store) GetCamera() (mgl32.Vec3, float32, float32) {
	var pos = mgl32.Vec3{0, 16, 0}
	var rx, ry float32
	s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(cameraBucket)
		value := bkt.Get(cameraBucket)
		if value == nil {
			return nil
		}
		buf := bytes.NewBuffer(value)
		binary.Read(buf, binary.LittleEndian, &pos)
		binary.Read(buf, binary.LittleEndian, &rx)
		binary.Read(buf, binary.LittleEndian, &ry)
		return nil
	})
	return pos, rx, ry
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

func (s *Store) Close() {
	s.db.Sync()
	s.db.Close()
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
