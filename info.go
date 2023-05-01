package gossip

import "github.com/ideaseeker/gossip/service"

type PeerInfo struct {
	Name string
}

func updatePeerData(data *service.PeerData, info *PeerInfo) {
	data.Name = info.Name
}

func peerDataToInfo(data *service.PeerData) *PeerInfo {
	return &PeerInfo{Name: data.Name}
}
