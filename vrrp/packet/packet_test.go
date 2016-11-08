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
	"bytes"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"l3/vrrp/config"
	"l3/vrrp/debug"
	"log/syslog"
	"net"
	"reflect"
	"testing"
	"utils/logging"
)

var testPktInfo *PacketInfo

var testEncodePkt = []byte{
	0x01, 0x00, 0x5e, 0x00, 0x00, 0x12, 0x00, 0x00, 0x5e, 0x00, 0x01, 0x01, 0x08, 0x00, 0x45, 0x00,
	0x00, 0x28, 0x00, 0x00, 0x00, 0x00, 0xff, 0x70, 0x1a, 0x8d, 0xc0, 0xa8, 0x00, 0x1e, 0xe0, 0x00,
	0x00, 0x12, 0x21, 0x01, 0x64, 0x01, 0x00, 0x01, 0xba, 0x52, 0xc0, 0xa8, 0x00, 0x01, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}
var testVrid = uint8(1)
var testPriority = uint8(100)
var testAdvInt = uint16(1)
var testVMac = "00:00:5e:00:01:01"
var testSrcIp = "192.168.0.30"
var testVip = "192.168.0.1"

func TestInit(t *testing.T) {
	testPktInfo = Init()
	if testPktInfo == nil {
		t.Error("failed to initialize packet info")
		return
	}
	var err error
	logger := new(logging.Writer)
	logger.MyComponentName = "VRRPD"
	logger.SysLogger, err = syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "VRRPTEST")
	if err != nil {
		t.Error("failed to initialize syslog:", err)
	} else {
		logger.MyLogLevel = 9 // trace level
		debug.SetLogger(logger)
	}
}

func TestEncodeV2(t *testing.T) {
	TestInit(t)
	pktInfo := &PacketInfo{
		Version:      config.VERSION2,
		Vrid:         testVrid,
		Priority:     testPriority,
		AdvertiseInt: testAdvInt,
		VirutalMac:   testVMac,
		IpAddr:       testSrcIp,
		Vip:          testVip,
	}
	encodedPkt := testPktInfo.Encode(pktInfo)
	if len(encodedPkt) != len(testEncodePkt) {
		t.Error("mis-match in length:", len(encodedPkt), len(testEncodePkt))
		return
	}
	if !bytes.Equal(encodedPkt, testEncodePkt) {
		t.Error("Failed to encode packet for pktInfo:", *pktInfo)
		t.Error("	testEncodePkt:", testEncodePkt)
		t.Error("	encoded Pkt:", encodedPkt)
		for idx, _ := range encodedPkt {
			if encodedPkt[idx] != testEncodePkt[idx] {
				t.Error("byte:", idx+1, "is not equal")
				t.Error(fmt.Sprintf("encoded Byte is:0x%x but wanted byte is:0x%x", encodedPkt[idx], testEncodePkt[idx]))
			}
		}
		return
	}
}

func TestDecodeV2(t *testing.T) {
	TestInit(t)
	p := gopacket.NewPacket(testEncodePkt, layers.LinkTypeEthernet, gopacket.Default)
	decodePkt := testPktInfo.Decode(p, config.VERSION2)
	if decodePkt == nil {
		t.Error("failed to decode packet")
		return
	}
	wantPktInfo := &PacketInfo{
		DstMac: VRRP_PROTOCOL_MAC,
		SrcMac: testVMac,
		IpAddr: testVip,
		DstIp:  VRRP_GROUP_IP,
		Hdr: &Header{
			Version:      config.VERSION2,
			Type:         VRRP_PKT_TYPE_ADVERTISEMENT,
			VirtualRtrId: testVrid,
			Priority:     testPriority,
			CountIPAddr:  1,
			Rsvd:         0,
			MaxAdverInt:  testAdvInt,
			CheckSum:     uint16(47698),
		},
	}
	wantPktInfo.Hdr.IpAddr = append(wantPktInfo.Hdr.IpAddr, net.ParseIP(testVip).To4())
	if !reflect.DeepEqual(wantPktInfo, decodePkt) {
		t.Error("failed to decode packet")
		t.Error("wantPktInfo header is:", *wantPktInfo.Hdr, "entire packet info:", wantPktInfo)
		t.Error("decodePktInfo header is:", *decodePkt.Hdr, "entire packet info:", decodePkt)
		return
	}
}
