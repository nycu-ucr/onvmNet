package onvmNet

// //#cgo CFLAGS: -I/home/ubuntu/openNetVM/onvm/onvm_nflib
// //#cgo CFLAGS: -I/home/ubuntu/openNetVM/onvm/lib
// //#cgo CFLAGS: -I/home/ubuntu/openNetVM/dpdk/x86_64-native-linuxapp-gcc/include
// #cgo CFLAGS: -I/home/ubuntu/openNetVM/onvm/lib
// #cgo LDFLAGS: /home/ubuntu/openNetVM/onvm/onvm_nflib/x86_64-native-linuxapp-gcc/libonvm.a
// #cgo LDFLAGS: /home/ubuntu/openNetVM/onvm/lib/x86_64-native-linuxapp-gcc/lib/libonvmhelper.a
// #cgo LDFLAGS: -Wl,--allow-multiple-definition
/*
#include <stdlib.h>
#include <rte_lcore.h>
#include <rte_common.h>
#include <rte_ip.h>
#include <rte_udp.h>
#include <rte_mbuf.h>
#include <onvm_nflib.h>
#include <onvm_pkt_helper.h>

static inline struct udp_hdr*
get_pkt_udp_hdr(struct rte_mbuf* pkt) {
    uint8_t* pkt_data = rte_pktmbuf_mtod(pkt, uint8_t*) + sizeof(struct ether_hdr) + sizeof(struct ipv4_hdr);
    return (struct udp_hdr*)pkt_data;
}
extern int onvmInit(struct onvm_nf_local_ctx *, int);

int ONVMSEND(char* bufPtr,struct onvm_nf_local_ctxconn nf_ctx,int id){
	return 10;
}
*/
import "C"

import (
	"fmt"
	"io/ioutil"
	"net"
	"reflect"
	"unsafe"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"gopkg.in/yaml.v2"
)

func Hi() {
  fmt.Println("Hi")
}

var udpChan = make(chan EthFrame, 1)
var handToReadChan = make(chan EthFrame,1)
var pktmbuf_pool *C.struct_rte_mempool
var pktCount int

type EthFrame struct {
  frame *C.struct_rte_mbuf
  frame_len int
}

type Config struct {
	ServiceID int32 `yaml:"serviceID"`
	IPIDMap   []struct {
		IP string `yaml:"IP"`
		ID int32  `yaml:"ID"`
	} `yaml:"IPIDMap"`
}

type OnvmConn struct {
	nf_ctx  *C.struct_onvm_nf_local_ctx
	udpChan chan EthFrame
}

//export Handler
func Handler(pkt *C.struct_rte_mbuf, meta *C.struct_onvm_pkt_meta,
	nf_local_ctx *C.struct_onvm_nf_local_ctx) int32 {
	pktCount++
	fmt.Println("packet received!")
	meta.action = C.ONVM_NF_ACTION_DROP

	udp_hdr := C.get_pkt_udp_hdr(pkt)

	if udp_hdr.dst_port == 2125 {
		udpChan <- EthFrame { pkt, int(C.rte_pktmbuf_data_len(pkt)) }
	}
	return 0
}

func (conn *OnvmConn) udpHandler() {
	var relayBuf EthFrame
	for {
		select {
		case relayBuf = <-udpChan:
			fmt.Println("Receive UDP")
			handToReadChan <- relayBuf
		}
	}
}

func ListenUDP(network string, laddr *net.UDPAddr) (*OnvmConn, error) {
	// Read Config
	var config Config
	if yamlFile, err := ioutil.ReadFile("udp.yaml"); err != nil {
		panic(err)
	} else {
		yaml.Unmarshal(yamlFile, config)
	}

	conn := &OnvmConn{}

	//C.onvmInit(conn.nf_ctx, C.int(config.ServiceID))
  C.onvmInit(conn.nf_ctx, C.int(1))

	pktmbuf_pool = C.rte_mempool_lookup(C.CString("MProc_pktmbuf_pool"))
	if pktmbuf_pool == nil {
		return nil, fmt.Errorf("pkt alloc from pool failed")
	}

	go conn.udpHandler()
	go C.onvm_nflib_run(conn.nf_ctx)

	fmt.Printf("ListenUDP: %s\n", network)
	return conn, nil
}

//func (conn * OnvmConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
//}

//func (conn * OnvmConn) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
//}

//func (conn * OnvmConn) LocalAddr() (laddr net.Addr) {
//}

func (conn *OnvmConn) Close() {

	C.onvm_nflib_stop(conn.nf_ctx)

	fmt.Println("Close onvm UDP")
}

func (conn * OnvmConn) WriteToUDP(b []byte ,addr * net.UDPAddr)(int,error){
    var success_send_len int
    var buffer_ptr *C.char //point to the head of byte data
    success_send_len = 0//???ONVM has functon to get it?
    tempBuffer:= marshalUDP(b,addr)
    buffer_ptr = getCPtrOfByteData(tempBuffer)
    //send the message to where???????
    success_send_len = C.ONVMSEND(buffer_ptr,conn.nf_ctx,10)//

    return success_send_len,nil
}

func (conn * OnvmConn) ReadFromUDP(b []byte)(int,*net.UDPAddr,error){
    buf := make([]byte,1500)
    var ethFame EthFrame
    ethFame = <- handToReadChan
    var recvLength int //????????onvm has function to get the length of buffer --> yes
    recvLength = ethFame.frame_len
    buf = C.GoByte(unsafe.Pointer(ethFame.frame))
    //C.memcpy(unsafe.Pointer(&b[0]),unsafe.Pointer(onvm_addr),1500)//??length not sure
    umsBuf,raddr := unMarshalUDP(buf)

    return recvLength,raddr,nil

}

func marshalUDP(b []byte,addr *net.UDPAddr)([]byte){
	fmt.Println("in marshal cap:",cap(b))
	ifi ,err :=net.InterfaceByName("en0")
	if err!=nil {
		panic(err)
	}
	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths: true,
	}

	ethlayer := &layers.Ethernet{
		SrcMAC: ifi.HardwareAddr,
		DstMAC: net.HardwareAddr{0,0,0,0,0,0},
		EthernetType: layers.EthernetTypeIPv4,
	}

	iplayer := &layers.IPv4{
		Version:uint8(4),
		SrcIP: addr.IP,
		DstIP: net.IP{192,168,0,1},
		TTL: 64,
		Protocol: layers.IPProtocolUDP,
	}

	udplayer := &layers.UDP{
		SrcPort: layers.UDPPort(addr.Port),
		DstPort: layers.UDPPort(80),
	}
	udplayer.SetNetworkLayerForChecksum(iplayer)
	err =gopacket.SerializeLayers(buffer,options,
		ethlayer,
		iplayer,
		udplayer,
		gopacket.Payload(b),
	)
	if err != nil{
		panic(err)
	}
	outgoingpacket := buffer.Bytes()
	return outgoingpacket

}

func getCPtrOfByteData(b []byte) *C.char{
	shdr := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	ptr := (*C.char)(unsafe.Pointer(shdr.Data))
	//runtime alive?
	return ptr
}

func unMarshalUDP(input []byte)(payLoad []byte,rAddr *net.UDPAddr){
	//Unmarshaludp header and get the information(ip port) from header
	var rPort int
	var rIp net.IP
	ethPacket :=gopacket.NewPacket(
		input,
		layers.LayerTypeEthernet,
		gopacket.Default)//this may be type zero copy

	ipLayer := ethPacket.Layer(layers.LayerTypeIPv4)

	if ipLayer != nil{
		ip,_:=ipLayer.(*layers.IPv4)
		rIp = ip.SrcIP
	}
	udpLayer := ethPacket.Layer(layers.LayerTypeUDP)
	if udpLayer != nil{
		udp,_ := udpLayer.(*layers.UDP)
		rPort = int(udp.SrcPort)
		payLoad = udp.Payload
	}

	rAddr = &net.UDPAddr{
		IP: rIp,
		Port: rPort,
	}

	return
}

func mMarshalUDP(b []byte,addr *net.UDPAddr)(output []byte){
    //wrapper payload with layer2 and layer3
    return
}
func uUnMarshalUDP(input []byte,output []byte)(*net.UDPAddr){
    //Unmarshaludp header and get the information(ip port) from header
    return nil
}

