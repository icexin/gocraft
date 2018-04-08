package main

import (
	"flag"
	"log"
	"net"
	"net/rpc"
	"strings"

	gocraft "github.com/icexin/gocraft-server/client"
	"github.com/icexin/gocraft-server/proto"
)

var (
	serverAddr = flag.String("s", "", "server address")

	client *gocraft.Client
)

func InitClient() error {
	if *serverAddr == "" {
		return nil
	}
	addr := *serverAddr
	if strings.Index(addr, ":") == -1 {
		addr += ":8421"
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	client = gocraft.NewClient()
	client.RegisterService("Block", &BlockService{})
	client.RegisterService("Player", &PlayerService{})
	client.Start(conn)
	return nil
}

func ClientFetchChunk(id Vec3, f func(bid Vec3, w int)) {
	if client == nil {
		return
	}
	req := proto.FetchChunkRequest{
		P:       id.X,
		Q:       id.Z,
		Version: store.GetChunkVersion(id),
	}
	rep := new(proto.FetchChunkResponse)
	err := client.Call("Block.FetchChunk", req, rep)
	if err == rpc.ErrShutdown {
		return
	}
	if err != nil {
		log.Panic(err)
	}
	for _, b := range rep.Blocks {
		f(Vec3{b[0], b[1], b[2]}, b[3])
	}
	if req.Version != rep.Version {
		store.UpdateChunkVersion(id, rep.Version)
	}
}

func ClientUpdateBlock(id Vec3, w int) {
	if client == nil {
		return
	}
	cid := id.Chunkid()
	req := &proto.UpdateBlockRequest{
		Id: client.ClientId,
		P:  cid.X,
		Q:  cid.Z,
		X:  id.X,
		Y:  id.Y,
		Z:  id.Z,
		W:  w,
	}
	rep := new(proto.UpdateBlockResponse)
	err := client.Call("Block.UpdateBlock", req, rep)
	if err == rpc.ErrShutdown {
		return
	}
	if err != nil {
		log.Panic(err)
	}
	store.UpdateChunkVersion(id.Chunkid(), rep.Version)
}

func ClientUpdatePlayerState(state PlayerState) {
	if client == nil {
		return
	}
	req := &proto.UpdateStateRequest{
		Id: client.ClientId,
	}
	s := &req.State
	s.X, s.Y, s.Z, s.Rx, s.Ry = state.X, state.Y, state.Z, state.Rx, state.Ry
	rep := new(proto.UpdateStateResponse)
	err := client.Call("Player.UpdateState", req, rep)
	if err == rpc.ErrShutdown {
		return
	}
	if err != nil {
		log.Panic(err)
	}

	for id, player := range rep.Players {
		game.playerRender.UpdateOrAdd(id, player)
	}
}

type BlockService struct {
}

func (s *BlockService) UpdateBlock(req *proto.UpdateBlockRequest, rep *proto.UpdateBlockResponse) error {
	log.Printf("rpc::UpdateBlock:%v", *req)
	bid := Vec3{req.X, req.Y, req.Z}
	game.world.UpdateBlock(bid, req.W)
	game.blockRender.DirtyChunk(bid.Chunkid())
	return nil
}

type PlayerService struct {
}

func (s *PlayerService) RemovePlayer(req *proto.RemovePlayerRequest, rep *proto.RemovePlayerResponse) error {
	game.playerRender.Remove(req.Id)
	return nil
}
