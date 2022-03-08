package p2p

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"

	"github.com/JM-Monteiro/torrent-client/peers"
	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/log"
	"github.com/davecgh/go-spew/spew"
)

func GetPeersFromDHT(infohash [20]byte, port int) ([]peers.Peer, error) {
	s, err := dht.NewServer(func() *dht.ServerConfig {
		sc := dht.NewDefaultServerConfig()
		sc.Logger = log.Default
		return sc
	}())
	if err != nil {
		log.Printf("error creating server: %s", err)
		return nil, err
	}
	defer s.Close()
	var wg sync.WaitGroup
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	addrs := make(map[[20]byte]map[string]struct{}, len(infohash))
	// PSA: Go sucks.
	a, err := s.Announce(infohash, port, false, func() (ret []dht.AnnounceOpt) {
		return
	}()...)
	if err != nil {
		log.Printf("error announcing %s: %s", infohash, err)
	}
	wg.Add(1)
	addrs[infohash] = make(map[string]struct{})
	go func(ih [20]byte) {
		defer wg.Done()
		defer a.Close()
	getPeers:
		for {
			select {
			case <-ctx.Done():
				a.StopTraversing()
				break getPeers
			case ps, ok := <-a.Peers:
				if !ok {
					break getPeers
				}
				for _, p := range ps.Peers {
					s := p.String()
					if _, ok := addrs[ih][s]; !ok {
						log.Printf("got peer %s for %x from %s", p, ih, ps.NodeInfo)
						addrs[ih][s] = struct{}{}
					}
				}
				if bf := ps.BFpe; bf != nil {
					log.Printf("%v claims %v peers for %x", ps.NodeInfo, bf.EstimateCount(), ih)
				}
				if bf := ps.BFsd; bf != nil {
					log.Printf("%v claims %v seeds for %x", ps.NodeInfo, bf.EstimateCount(), ih)
				}
			}
		}
		log.Levelf(log.Debug, "finishing traversal")
		<-a.Finished()
		//log.Printf("%v contacted %v nodes", a, a.NumContacted())
	}(infohash)

	wg.Wait()
	spew.Dump(s.Stats())

	peerList := make([]peers.Peer, 0)
	ips := make(map[string]struct{}, len(addrs[infohash]))
	for s := range addrs[infohash] {
		ip, port, err := net.SplitHostPort(s)
		if err != nil {
			log.Printf("error parsing addr: %s", err)
		}
		ips[ip] = struct{}{}
		ipnet := net.ParseIP(ip)
		portN, _ := strconv.ParseUint(port, 0, 16)
		peerList = append(peerList, peers.Peer{ipnet, uint16(portN), "UDP"})
	}
	log.Printf("%x: %d addrs %d distinct ips", infohash, len(addrs[infohash]), len(ips))

	return peerList, nil
}
