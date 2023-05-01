//go:build !solution

package gossip

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/ideaseeker/gossip/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PeerConfig struct {
	SelfAddr   string
	PingPeriod time.Duration
}

type Peer struct {
	service.UnimplementedGossipServiceServer

	config PeerConfig

	ctx    context.Context
	cancel context.CancelFunc

	connections *Connections
}

func NewPeer(config PeerConfig) *Peer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Peer{
		config:      config,
		ctx:         ctx,
		cancel:      cancel,
		connections: NewConnections(config.SelfAddr),
	}
}

func (p *Peer) Addr() string {
	return p.config.SelfAddr
}

func (p *Peer) GetMembers() map[string]PeerInfo {
	allData := p.connections.GetAllData()

	members := make(map[string]PeerInfo, len(allData))
	for addr, data := range allData {
		members[addr] = peerDataToInfo(data)
	}

	return members
}

func (p *Peer) UpdateSelfInfo(info *PeerInfo) {
	p.connections.UpdateSelfInfo(info)
	log.Println(p.Addr(), "update info:", info)
}

func (p *Peer) AddSeed(seed string) {
	conn, err := p.openConn(seed)
	if err != nil {
		log.Println(p.Addr(), "cannot open connection to", seed, "because of", err)
		return
	}

	p.connections.AddSeed(seed, conn)
	log.Println(p.Addr(), "added a new seed", seed)
}

func (p *Peer) Run() {
	log.Println(p.Addr(), "started")
	ticker := time.NewTicker(p.config.PingPeriod)
	for {
		select {
		case <-p.ctx.Done():
			log.Println(p.Addr(), "client stopped because of", p.ctx.Err())
			return
		case <-ticker.C:
			p.sendInfoToSomeone()
		}
	}
}

func (p *Peer) Stop() {
	p.cancel()
	log.Println(p.Addr(), "cancelled context")

	p.connections.Close()
	log.Println(p.Addr(), "close all connections")
}

// gRPC server implementation

func (p *Peer) Ping(context.Context, *service.PingMessage) (*service.PongMessage, error) {
	return &service.PongMessage{}, nil
}

func (p *Peer) ShareData(stream service.GossipService_ShareDataServer) error {
	otherData, err := p.recvAllInfo(stream.Recv)
	if err != nil {
		return err
	}
	p.updateWithOtherData(otherData)
	log.Println(p.Addr(), "received other data")

	if err = p.sendAllInfo(stream.Send); err != nil {
		return err
	}
	log.Println(p.Addr(), "sent self data")

	return nil
}

// gRPC client implementation

func (p *Peer) openConn(seed string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(seed, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	if err = p.pingConn(conn); err != nil {
		log.Println(p.Addr(), "failed to ping", seed, "because of", err)
		_ = conn.Close()
		return nil, err
	}

	return conn, nil
}

func (p *Peer) pingConn(conn *grpc.ClientConn) error {
	client := service.NewGossipServiceClient(conn)

	_, err := client.Ping(p.ctx, &service.PingMessage{})
	if err != nil {
		return err
	}

	return nil
}

func (p *Peer) sendInfoToSomeone() {
	seed, conn, err := p.connections.GetRandomConn()
	if err != nil {
		return
	}

	log.Println(p.Addr(), "chose", seed, "to send info")
	if err = p.sendDataToConn(conn); err != nil {
		p.connections.DeleteSeed(seed)
		log.Println(p.Addr(), "deleted", seed, "because of", err)
	}
}

func (p *Peer) sendDataToConn(conn *grpc.ClientConn) error {
	client := service.NewGossipServiceClient(conn)
	stream, err := client.ShareData(p.ctx)
	if err != nil {
		log.Println(p.Addr(), "cannot obtain share-stream because of", err)
		return err
	}

	if err = p.sendAllInfo(stream.Send); err != nil {
		return err
	}
	_ = stream.CloseSend()
	log.Println(p.Addr(), "sent self data")

	otherData, err := p.recvAllInfo(stream.Recv)
	if err != nil {
		return err
	}
	p.updateWithOtherData(otherData)
	log.Println(p.Addr(), "received other data")

	return nil
}

// internal

func (p *Peer) sendAllInfo(send func(*service.PeerData) error) error {
	for _, info := range p.connections.GetAllData() {
		if p.ctx.Err() != nil {
			log.Println(p.Addr(), "context cancelled during sendAllInfo")
			return nil
		}
		if err := send(info); err != nil {
			log.Println(p.Addr(), "got", err, "during sendAllInfo")
			return err
		}
	}

	return nil
}

func (p *Peer) recvAllInfo(recv func() (*service.PeerData, error)) (map[string]*service.PeerData, error) {
	result := make(map[string]*service.PeerData)

	for {
		if p.ctx.Err() != nil {
			log.Println(p.Addr(), "context cancelled during recvAllInfo")
			return nil, nil
		}

		data, err := recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println(p.Addr(), "got", err, "during recvAllInfo")
			return nil, err
		}

		result[data.GetAddr()] = data
	}

	return result, nil
}

func (p *Peer) updateWithOtherData(otherData map[string]*service.PeerData) {
	for seed, data := range otherData {
		exists := p.connections.UpsertData(seed, data)
		if !exists {
			p.AddSeed(seed)
		}
	}
}
