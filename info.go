package gossip

import "github.com/mmvpm/gossip/service"

type PeerInfo struct {
	Name string
}

func peerDataToInfo(data *service.PeerData) PeerInfo {
	return PeerInfo{Name: data.Name}
}
