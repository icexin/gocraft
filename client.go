package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"
)

type Client struct {
	lock sync.Mutex
	conn net.Conn
	r    *bufio.Reader
}

func NewClient(addr string) *Client {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	return &Client{
		conn: conn,
		r:    bufio.NewReader(conn),
	}
}

func (c *Client) readCommand() string {
	line, err := c.r.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	return line[:len(line)-1]
}

func (c *Client) FetchChunk(id Vec3) map[Vec3]int {
	c.lock.Lock()
	defer c.lock.Unlock()
	fmt.Fprintf(c.conn, "C,%d,%d,123\r\n", id.X, id.Z)
	m := make(map[Vec3]int)
	for {
		cmd := c.readCommand()
		if cmd[0] == 'C' {
			break
		}
		if cmd[0] != 'B' {
			continue
		}
		var p, q, x, y, z, w int
		fmt.Sscanf(cmd, "B,%d,%d,%d,%d,%d,%d", &p, &q, &x, &y, &z, &w)
		block := Vec3{x, y, z}
		if block.Chunkid() != id {
			// log.Printf("block %v chunk %v, %v", block, block.Chunkid(), id)
			continue
		}
		if w == 0 {
			continue
		}
		m[block] = w
	}
	return m
}
