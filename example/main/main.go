package main

import (
	"log"
	"time"

	"github.com/mmvpm/gossip"
)

const pingPeriod = 100 * time.Millisecond

func main() {
	config := gossip.PeerConfig{
		SelfAddr:   "127.0.0.1:0", // any free port
		PingPeriod: pingPeriod,
	}

	peer0, stop0, _ := gossip.SpawnNewPeer(config)
	peer1, stop1, _ := gossip.SpawnNewPeer(config)

	peer0.UpdateSelfInfo(&gossip.PeerInfo{Name: "peer0"})
	peer1.UpdateSelfInfo(&gossip.PeerInfo{Name: "peer1"})

	peer0.AddSeed(peer1.Addr())

	time.Sleep(pingPeriod * 10) // gossip sync

	log.Println("peer0 members:", peer0.GetMembers())
	log.Println("peer1 members:", peer1.GetMembers())

	stop0()

	time.Sleep(pingPeriod * 10) // gossip sync

	log.Println("peer1 members:", peer1.GetMembers())

	stop1()
}
