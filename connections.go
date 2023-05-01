package gossip

import (
	"container/list"
	"errors"
	"sync"

	"github.com/ideaseeker/gossip/service"
	"google.golang.org/grpc"
)

type Connections struct {
	mu           sync.RWMutex
	selfAddr     string
	lruSeedOrder *list.List
	seedToData   map[string]*service.PeerData
	connections  map[string]*grpc.ClientConn
}

func NewConnections(addr string) *Connections {
	return &Connections{
		mu:           sync.RWMutex{},
		selfAddr:     addr,
		lruSeedOrder: list.New(),
		seedToData:   map[string]*service.PeerData{addr: newPeerData(addr)},
		connections:  make(map[string]*grpc.ClientConn),
	}
}

func (c *Connections) AddSeed(seed string, conn *grpc.ClientConn) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.connections[seed]; !exists {
		c.lruSeedOrder.PushFront(seed)
		c.seedToData[seed] = newPeerData(seed)
		c.connections[seed] = conn
	}
}

func (c *Connections) DeleteSeed(seed string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.connections[seed].Close()
	delete(c.connections, seed)

	c.seedToData[seed].Deleted = true
	c.seedToData[seed].Version += 1
}

func (c *Connections) UpdateSelfInfo(info *PeerInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	updatePeerData(c.seedToData[c.selfAddr], info)
	c.seedToData[c.selfAddr].Version += 1
}

func (c *Connections) UpsertData(seed string, newData *service.PeerData) (alreadyExists bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	curData, exists := c.seedToData[seed]
	if !exists || newData.Version > curData.Version {
		c.seedToData[seed] = newData
	}

	return exists
}

func (c *Connections) GetAllData() map[string]*service.PeerData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	members := make(map[string]*service.PeerData, len(c.connections)+1)
	for addr, data := range c.seedToData {
		if !data.Deleted {
			members[addr] = data
		}
	}

	return members
}

func (c *Connections) GetRandomConn() (string, *grpc.ClientConn, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.connections) == 0 {
		return "", nil, errors.New("error: no connections available")
	}

	for {
		frontElem := c.lruSeedOrder.Front()
		seed := frontElem.Value.(string)
		if conn, exists := c.connections[seed]; exists {
			c.lruSeedOrder.MoveToBack(frontElem)
			return seed, conn, nil
		} else {
			c.lruSeedOrder.Remove(frontElem)
		}
	}
}

func (c *Connections) Close() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, conn := range c.connections {
		_ = conn.Close()
	}
}

// internal

func newPeerData(seed string) *service.PeerData {
	return &service.PeerData{Addr: seed}
}
