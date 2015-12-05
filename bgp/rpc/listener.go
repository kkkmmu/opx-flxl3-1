// server.go
package rpc

import (
	"bgpd"
	"fmt"
	"l3/bgp/config"
	"l3/bgp/server"
	"log/syslog"
	"net"
)

type PeerConfigCommands struct {
	IP      net.IP
	Command int
}

type BGPHandler struct {
	PeerCommandCh chan PeerConfigCommands
	server        *server.BGPServer
	logger        *syslog.Writer
}

func NewBGPHandler(server *server.BGPServer, logger *syslog.Writer) *BGPHandler {
	h := new(BGPHandler)
	h.PeerCommandCh = make(chan PeerConfigCommands)
	h.server = server
	h.logger = logger
	return h
}

func (h *BGPHandler) CreateBGPGlobal(bgpGlobal *bgpd.BGPGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create global config attrs:", bgpGlobal))
	if bgpGlobal.RouterId == "localhost" {
		bgpGlobal.RouterId = "127.0.0.1"
	}

	ip := net.ParseIP(bgpGlobal.RouterId)
	if ip == nil {
		h.logger.Info(fmt.Sprintln("CreateBGPGlobal - IP is not valid:", bgpGlobal.RouterId))
		return false, nil
	}

	gConf := config.GlobalConfig{
		AS:       uint32(bgpGlobal.AS),
		RouterId: ip,
	}
	h.server.GlobalConfigCh <- gConf
	return true, nil
}

func (h *BGPHandler) UpdateBGPGlobal(bgpGlobal *bgpd.BGPGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Update global config attrs:", bgpGlobal))
	return true, nil
}

func (h *BGPHandler) DeleteBGPGlobal(bgpGlobal *bgpd.BGPGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete global config attrs:", bgpGlobal))
	return true, nil
}

func (h *BGPHandler) CreateBGPNeighbor(bgpNeighbor *bgpd.BGPNeighbor) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create peer attrs:", bgpNeighbor))
	ip := net.ParseIP(bgpNeighbor.NeighborAddress)
	if ip == nil {
		h.logger.Info(fmt.Sprintln("CreatePeer - IP is not valid:", bgpNeighbor.NeighborAddress))
	}
	pConf := config.NeighborConfig{
		PeerAS:          uint32(bgpNeighbor.PeerAS),
		LocalAS:         uint32(bgpNeighbor.LocalAS),
		Description:     bgpNeighbor.Description,
		NeighborAddress: ip,
	}
	h.server.AddPeerCh <- pConf
	return true, nil
}

func (h *BGPHandler) UpdateBGPNeighbor(bgpNeighbor *bgpd.BGPNeighbor) (bool, error) {
	h.logger.Info(fmt.Sprintln("Update peer attrs:", bgpNeighbor))
	return true, nil
}

func (h *BGPHandler) DeleteBGPNeighbor(bgpNeighbor *bgpd.BGPNeighbor) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete peer attrs:", bgpNeighbor))
	ip := net.ParseIP(bgpNeighbor.NeighborAddress)
	if ip == nil {
		h.logger.Info(fmt.Sprintln("CreatePeer - IP is not valid:", bgpNeighbor.NeighborAddress))
	}
	pConf := config.NeighborConfig{
		PeerAS:          uint32(bgpNeighbor.PeerAS),
		LocalAS:         uint32(bgpNeighbor.LocalAS),
		Description:     bgpNeighbor.Description,
		NeighborAddress: ip,
	}
	h.server.RemPeerCh <- pConf
	return true, nil
}

func (h *BGPHandler) PeerCommand(in *PeerConfigCommands, out *bool) error {
	h.PeerCommandCh <- *in
	h.logger.Info(fmt.Sprintln("Good peer command:", in))
	*out = true
	return nil
}
