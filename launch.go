package gossip

import (
	"net"

	"github.com/ideaseeker/gossip/service"
	"google.golang.org/grpc"
)

func SpawnNewPeer(config PeerConfig) (peer *Peer, stop func(), err error) {
	listener, err := net.Listen("tcp", config.SelfAddr)
	if err != nil {
		return
	}

	peer = NewPeer(PeerConfig{
		SelfAddr:   listener.Addr().String(),
		PingPeriod: config.PingPeriod,
	})

	server := grpc.NewServer()
	service.RegisterGossipServiceServer(server, peer)

	go func() { _ = server.Serve(listener) }()
	go peer.Run()

	stop = func() {
		server.Stop()
		peer.Stop()
	}

	return
}
