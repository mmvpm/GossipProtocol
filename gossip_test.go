package gossip_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/mmvpm/gossip"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

const (
	pingPeriod = time.Millisecond * 50
	waitPeriod = pingPeriod * 10
)

type testEnv struct {
	newPeer func() (*gossip.Peer, func())
}

func newTestEnv(t *testing.T) *testEnv {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	return &testEnv{
		newPeer: func() (*gossip.Peer, func()) {
			config := gossip.PeerConfig{
				SelfAddr:   "127.0.0.1:0", // any free port
				PingPeriod: pingPeriod,
			}

			peer, stop, err := gossip.SpawnNewPeer(config)
			require.NoError(t, err)

			t.Cleanup(stop)

			return peer, stop
		},
	}
}

func TestSinglePeer(t *testing.T) {
	env := newTestEnv(t)

	peer0, _ := env.newPeer()

	members := peer0.GetMembers()
	require.Contains(t, members, peer0.Addr())

	peer0.UpdateSelfInfo(&gossip.PeerInfo{Name: "peer0"})

	members = peer0.GetMembers()
	require.Contains(t, members, peer0.Addr())
	require.Equal(t, "peer0", members[peer0.Addr()].Name)

	deadPeer, stopDeadPeer := env.newPeer()
	stopDeadPeer()

	peer0.AddSeed(deadPeer.Addr())
	time.Sleep(waitPeriod)

	require.Len(t, peer0.GetMembers(), 1)
}

func TestDeadPeer(t *testing.T) {
	env := newTestEnv(t)

	peer0, _ := env.newPeer()
	deadPeer, stopDeadPeer := env.newPeer()
	stopDeadPeer()

	peer0.AddSeed(deadPeer.Addr())
	time.Sleep(waitPeriod)

	require.NotContains(t, peer0.GetMembers(), deadPeer.Addr())
}

func TestTwoPeers(t *testing.T) {
	env := newTestEnv(t)

	peer0, _ := env.newPeer()
	peer1, stop1 := env.newPeer()

	peer0.AddSeed(peer1.Addr())
	time.Sleep(waitPeriod)

	members0 := peer0.GetMembers()
	require.Contains(t, members0, peer0.Addr())
	require.Contains(t, members0, peer1.Addr())

	members1 := peer1.GetMembers()
	require.Contains(t, members1, peer0.Addr())
	require.Contains(t, members1, peer1.Addr())

	peer0.UpdateSelfInfo(&gossip.PeerInfo{Name: "peer0"})
	time.Sleep(waitPeriod)
	require.Equal(t, "peer0", peer1.GetMembers()[peer0.Addr()].Name)

	peer1.UpdateSelfInfo(&gossip.PeerInfo{Name: "peer1"})
	time.Sleep(waitPeriod)
	require.Equal(t, "peer1", peer0.GetMembers()[peer1.Addr()].Name)

	stop1()
	time.Sleep(waitPeriod * 5)

	members0 = peer0.GetMembers()
	require.NotContains(t, members0, peer1.Addr())
}

func TestSeveralPeers(t *testing.T) {
	env := newTestEnv(t)

	mainPeer, stopMainPeer := env.newPeer()

	var peers []*gossip.Peer
	names := map[string]string{}

	for i := 0; i < 10; i++ {
		peer, _ := env.newPeer()
		peers = append(peers, peer)

		name := fmt.Sprintf("peer%d", i)
		peer.UpdateSelfInfo(&gossip.PeerInfo{Name: name})
		names[peer.Addr()] = name

		peer.AddSeed(mainPeer.Addr())
	}

	time.Sleep(waitPeriod)
	stopMainPeer()
	time.Sleep(waitPeriod * 10)

	for _, peer := range peers {
		members := peer.GetMembers()
		require.NotContains(t, members, mainPeer.Addr())
		for addr, name := range names {
			require.Contains(t, members, addr)
			require.Equal(t, name, members[addr].Name)
		}
	}

	peers[0].UpdateSelfInfo(&gossip.PeerInfo{Name: "main"})
	time.Sleep(waitPeriod)

	for _, peer := range peers {
		members := peer.GetMembers()

		require.Contains(t, members, peers[0].Addr())
		require.Equal(t, "main", members[peers[0].Addr()].Name)
	}
}

func TestGroups(t *testing.T) {
	env := newTestEnv(t)

	aSize, bSize := 1, 1
	peerA, _ := env.newPeer()
	peerB, _ := env.newPeer()

	for i := 0; i < 10; i++ {
		peer, _ := env.newPeer()
		if rand.Int()%2 == 0 {
			peer.AddSeed(peerA.Addr())
			aSize += 1
		} else {
			peer.AddSeed(peerB.Addr())
			bSize += 1
		}
	}

	time.Sleep(waitPeriod)
	require.Len(t, peerA.GetMembers(), aSize)
	require.Len(t, peerB.GetMembers(), bSize)

	peerA.AddSeed(peerB.Addr())
	time.Sleep(waitPeriod)
	require.Len(t, peerA.GetMembers(), aSize+bSize)
	require.Len(t, peerB.GetMembers(), aSize+bSize)
}
