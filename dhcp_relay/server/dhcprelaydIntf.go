// Dhcp Relay Agent Interface Handling
package relayServer

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"dhcprelayd"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	nanomsg "github.com/op/go-nanomsg"
	"net"
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

type DHCPRELAYClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type AsicdClient struct {
	DHCPRELAYClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

func DhcpRelayAgentListenAsicUpdate(address string) error {
	var err error
	logger.Info("DRA: setting up asicd update listener")
	if asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		logger.Err(fmt.Sprintln("DRA: Failed to create ASIC subscribe socket, error:", err))
		return err
	}

	if err = asicdSubSocket.Subscribe(""); err != nil {
		logger.Err(fmt.Sprintln("DRA:Failed to subscribe to \"\" on ASIC subscribe socket, error:", err))
		return err
	}

	if _, err = asicdSubSocket.Connect(address); err != nil {
		logger.Err(fmt.Sprintln("DRA: Failed to connect to ASIC publisher socket, address:", address, "error:", err))
		return err
	}

	logger.Info(fmt.Sprintln("DRA: Connected to ASIC publisher at address:", address))
	if err = asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		logger.Err(fmt.Sprintln("DRA: Failed to set the buffer size for ASIC publisher socket, error:", err))
		return err
	}
	logger.Info("DRA: asicd update listener is set")
	return nil
}

func DhcpRelayAgentUpdateVlanInfo(vlanNotifyMsg asicdConstDefs.VlanNotifyMsg, msgType uint8) {
	logger.Info(fmt.Sprintln("DRA: Vlan update message for ",
		vlanNotifyMsg.VlanName, "vlan id is ", vlanNotifyMsg.VlanId))
	var linuxInterface *net.Interface
	var err error
	linuxInterface, err = net.InterfaceByName(vlanNotifyMsg.VlanName)
	if err != nil {
		logger.Err(fmt.Sprintln("DRA: getting interface by name failed", err))
		return
	}
	if msgType == asicdConstDefs.NOTIFY_VLAN_CREATE { // Create Vlan
		dhcprelayLogicalIntfId2LinuxIntId[linuxInterface.Index] =
			int32(vlanNotifyMsg.VlanId)
	} else { // Delete interface id
		delete(dhcprelayLogicalIntfId2LinuxIntId, linuxInterface.Index)
	}
}

func DhcpRelayAgentUpdateIntfPortMap(msg asicdConstDefs.IPv4IntfNotifyMsg, msgType uint8) {
	logicalId := int32(asicdConstDefs.GetIntfIdFromIfIndex(msg.IfIndex))
	intfId := msg.IfIndex
	logger.Info(fmt.Sprintln("DRA: Got a ipv4 interface notification for:", msgType,
		"for If Id:", intfId, "original id is", msg.IfIndex))
	if msgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE {
		dhcprelayLogicalIntf2IfIndex[logicalId] = intfId
		// @TODO: fix netmask later on...
		// Init DRA Global Handling for new interface....
		// 192.168.1.1/24 -> ip: 192.168.1.1  net: 192.168.1.0/24
		DhcpRelayAgentInitGblHandling(intfId, false)
		DhcpRelayAgentInitIntfState(intfId)
		gblEntry := dhcprelayGblInfo[intfId]
		ip, ipnet, err := net.ParseCIDR(msg.IpAddr)
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: Parsing ipadd and netmask failed:", err))
			return
		}
		gblEntry.IpAddr = ip.String()
		gblEntry.Netmask = ipnet.IP.String()
		dhcprelayGblInfo[intfId] = gblEntry
		logger.Info(fmt.Sprintln("DRA: Added interface:", intfId, " Ip address:",
			gblEntry.IpAddr, " netmask:", gblEntry.Netmask))
	} else {
		logger.Info(fmt.Sprintln("DRA: deleteing interface", intfId))
		delete(dhcprelayGblInfo, intfId)
	}
}

func DhcpRelayAgentUpdateL3IntfStateChange(msg asicdConstDefs.L3IntfStateNotifyMsg) {
	if msg.IfState == asicdConstDefs.INTF_STATE_UP {
		logger.Info(fmt.Sprintln("DRA: Got intf state up notification"))

	} else if msg.IfState == asicdConstDefs.INTF_STATE_DOWN {
		logger.Info(fmt.Sprintln("DRA: Got intf state down notification"))

	}
}
func DhcpRelayAsicdSubscriber() {
	for {
		logger.Info("DRA: Read on Asic Subscriber socket....")
		rxBuf, err := asicdSubSocket.Recv(0)
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: Recv on asicd Subscriber socket failed with error:", err))
			continue
		}
		//logger.Info(fmt.Sprintln("DRA: asicd Subscriber recv returned:", rxBuf))
		var msg asicdConstDefs.AsicdNotification
		err = json.Unmarshal(rxBuf, &msg)
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: Unable to Unmarshal asicd msg:", msg.Msg))
			continue
		}
		if msg.MsgType == asicdConstDefs.NOTIFY_VLAN_CREATE ||
			msg.MsgType == asicdConstDefs.NOTIFY_VLAN_DELETE {
			//Vlan Create Msg
			var vlanNotifyMsg asicdConstDefs.VlanNotifyMsg
			err = json.Unmarshal(msg.Msg, &vlanNotifyMsg)
			if err != nil {
				logger.Err(fmt.Sprintln("DRA: Unable to unmashal vlanNotifyMsg:", msg.Msg))
				return
			}
			DhcpRelayAgentUpdateVlanInfo(vlanNotifyMsg, msg.MsgType)
		} else if msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE ||
			msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_DELETE {
			var ipv4IntfNotifyMsg asicdConstDefs.IPv4IntfNotifyMsg
			err = json.Unmarshal(msg.Msg, &ipv4IntfNotifyMsg)
			if err != nil {
				logger.Err(fmt.Sprintln("DRA: Unable to Unmarshal ipv4IntfNotifyMsg:", msg.Msg))
				continue
			}
			DhcpRelayAgentUpdateIntfPortMap(ipv4IntfNotifyMsg, msg.MsgType)
		} else if msg.MsgType == asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE {
			//INTF_STATE_CHANGE
			var l3IntfStateNotifyMsg asicdConstDefs.L3IntfStateNotifyMsg
			err = json.Unmarshal(msg.Msg, &l3IntfStateNotifyMsg)
			if err != nil {
				logger.Err(fmt.Sprintln("DRA: unable to Unmarshal l3 intf state change:", msg.Msg))
				continue
			}
			DhcpRelayAgentUpdateL3IntfStateChange(l3IntfStateNotifyMsg)
		}
	}
}

func DhcpRelayAgentAllocateMemory() {
	// Allocate memory for Global Info
	dhcprelayGblInfo = make(map[int32]DhcpRelayAgentGlobalInfo, 50)
	// Interface State Maps
	dhcprelayIntfStateMap = make(map[int32]dhcprelayd.DhcpRelayIntfState, 50)
	dhcprelayIntfServerStateMap = make(map[string]dhcprelayd.DhcpRelayIntfServerState, 50)
	// Interface State Slice
	dhcprelayIntfStateSlice = []int32{}
	dhcprelayIntfServerStateSlice = []string{}
	// Allocate memory for Linux ID ---> Logical Id mapping
	dhcprelayLogicalIntfId2LinuxIntId = make(map[int]int32, 30)
	// Logical Id to Unique If Index, for e.g vlan 9
	// 9 will map to 33554441
	dhcprelayLogicalIntf2IfIndex = make(map[int32]int32, 10)

	// Ipv4Intf map
	dhcprelayIntfIpv4Map = make(map[int32]asicdServices.IPv4Intf, 3)
}

func DhcpRelayAgentGetPortList() {
	currMarker := int64(asicdConstDefs.MIN_SYS_PORTS)
	more := false
	objCount := 0
	count := 10
	for {
		bulkInfo, err := asicdClient.ClientHdl.GetBulkPortState(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: getting bulk port config"+
				" from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			var portNum int32
			portNum = bulkInfo.PortStateList[i].IfIndex
			DhcpRelayAgentInitGblHandling(portNum, false)
			DhcpRelayAgentInitIntfState(portNum)
		}
		if more == false {
			break
		}
	}
}

func DhcpRelayAgentGetVlanList() {
	objCount := 0
	var currMarker int64
	more := false
	count := 10
	for {
		bulkInfo, err := asicdClient.ClientHdl.GetBulkVlanState(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: getting bulk vlan config",
				"from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			DhcpRelayAgentInitVlanInfo(bulkInfo.VlanStateList[i].VlanName,
				bulkInfo.VlanStateList[i].VlanId)
		}
		if more == false {
			break
		}
	}
}

func DhcpRelayAgentGetIpv4IntfList() {
	objCount := 0
	var currMarker int64
	more := false
	count := 10
	for {
		bulkInfo, err := asicdClient.ClientHdl.GetBulkIPv4Intf(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			logger.Err(fmt.Sprintln("DRA: getting bulk vlan config",
				"from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			obj := dhcprelayIntfIpv4Map[bulkInfo.IPv4IntfList[i].IfIndex]
			obj.IfIndex = bulkInfo.IPv4IntfList[i].IfIndex
			obj.IpAddr = bulkInfo.IPv4IntfList[i].IpAddr
			dhcprelayIntfIpv4Map[bulkInfo.IPv4IntfList[i].IfIndex] = obj
		}
		if more == false {
			break
		}
	}
}

/*
 * DhcpRelayInitPortParams:
 *	    API to handle initialization of port parameter
 */
func DhcpRelayInitPortParams() error {
	logger.Info("DRA: initializing Port Parameters & Global Init")
	if !asicdClient.IsConnected {
		logger.Info("DRA: is not connected to asicd.... is it bad?")
		return nil
	}
	err := DhcpRelayAgentListenAsicUpdate(asicdConstDefs.PUB_SOCKET_ADDR)
	if err == nil {
		// Asicd subscriber thread
		go DhcpRelayAsicdSubscriber()
	}
	logger.Info("DRA calling asicd for port config")
	// Get Port State Information
	DhcpRelayAgentGetPortList()
	// Get Vlans Information
	logger.Info("DRA: Initializing Vlan Info (if any)")
	DhcpRelayAgentGetVlanList()

	logger.Info("DRA: initialized Port Parameters & Global Info successfully")
	return nil
}
