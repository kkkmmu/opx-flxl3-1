//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  |
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  |
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   |
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  |
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__|
//
package packet

import (
	"encoding/binary"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"l3/vrrp/common"
	_ "l3/vrrp/debug"
	"net"
)

/*
Octet Offset--> 0                   1                   2                   3
 |		0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 |		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 V		|                    IPv4 Fields or IPv6 Fields                 |
		...                                                             ...
		|                                                               |
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 0		|Version| Type  | Virtual Rtr ID|   Priority    |Count IPvX Addr|
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 4		|(rsvd) |     Max Adver Int     |          Checksum             |
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 8		|                                                               |
		+                                                               +
12		|                       IPvX Address(es)                        |
		+                                                               +
..		+                                                               +
		+                                                               +
		+                                                               +
		|                                                               |
		+                                                               +
		|                                                               |
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/
func EncodeHeader(hdr *Header) ([]byte, uint16) {
	pktLen := VRRP_HEADER_SIZE_EXCLUDING_IPVX + (hdr.CountIPAddr * 4)
	if pktLen < VRRP_HEADER_MIN_SIZE {
		pktLen = VRRP_HEADER_MIN_SIZE
	}
	bytes := make([]byte, pktLen)
	bytes[0] = (hdr.Version << 4) | hdr.Type
	bytes[1] = hdr.VirtualRtrId
	bytes[2] = hdr.Priority
	bytes[3] = hdr.CountIPAddr
	rsvdAdver := (uint16(hdr.Rsvd) << 13) | hdr.MaxAdverInt
	binary.BigEndian.PutUint16(bytes[4:], rsvdAdver)
	binary.BigEndian.PutUint16(bytes[6:8], hdr.CheckSum)
	baseIpByte := 8
	for i := 0; i < int(hdr.CountIPAddr); i++ {
		copy(bytes[baseIpByte:(baseIpByte+4)], hdr.IpAddr[i].To4())
		baseIpByte += 4
	}
	// Create Checksum for the header and store it
	binary.BigEndian.PutUint16(bytes[6:8], computeChecksum(hdr.Version, bytes))
	return bytes, uint16(pktLen)
}

func CreateHeader(pInfo *PacketInfo) ([]byte, uint16) {
	hdr := Header{
		Version:      pInfo.Version,
		Type:         VRRP_PKT_TYPE_ADVERTISEMENT,
		VirtualRtrId: pInfo.Vrid,
		Priority:     pInfo.Priority,
		CountIPAddr:  1, // FIXME for more than 1 vip
		Rsvd:         VRRP_RSVD,
		MaxAdverInt:  pInfo.AdvertiseInt,
		CheckSum:     VRRP_HDR_CREATE_CHECKSUM,
	}
	ip, _, _ := net.ParseCIDR(pInfo.Vip)
	if ip == nil {
		// means that we got absolute ip address as part of packet information
		ip = net.ParseIP(pInfo.Vip)
	}
	hdr.IpAddr = append(hdr.IpAddr, ip)
	//debug.Logger.Debug("Vrrp Header:", hdr)
	return EncodeHeader(&hdr)
}

func (p *PacketInfo) Encode(pInfo *PacketInfo) []byte {
	payload, hdrLen := CreateHeader(pInfo)
	// Ethernet Layer
	srcMAC, _ := net.ParseMAC(pInfo.VirutalMac)
	dstMAC, _ := net.ParseMAC(VRRP_PROTOCOL_MAC)
	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}
	//debug.Logger.Debug("(dmac, smac):(", dstMAC.String(), ",", srcMAC.String(), ")")
	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	sip, _, _ := net.ParseCIDR(pInfo.IpAddr)
	if sip == nil {
		sip = net.ParseIP(pInfo.IpAddr)
	}
	// IPvX Layer
	switch pInfo.Version {
	case common.VERSION2:
		dip := net.ParseIP(VRRP_V4_GROUP_IP)
		ipv4 := &layers.IPv4{
			Version:  uint8(4),
			IHL:      uint8(VRRP_IPV4_HEADER_MIN_SIZE),
			Protocol: layers.IPProtocol(VRRP_PROTO_ID),
			Length:   uint16(VRRP_IPV4_HEADER_MIN_SIZE + hdrLen),
			TTL:      uint8(VRRP_TTL),
			SrcIP:    sip,
			DstIP:    dip,
		}
		//debug.Logger.Debug("ipv4 information is:", *ipv4)
		gopacket.SerializeLayers(buffer, options, eth, ipv4, gopacket.Payload(payload))

	case common.VERSION3:
		// @TODO: need to create ipv6 information after reading rfc
	}

	return buffer.Bytes()
}
