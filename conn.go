package onvmNet

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
//wrapper for c macro
static inline int pktmbuf_data_len_wrapper(struct rte_mbuf* pkt){
	return rte_pktmbuf_data_len(pkt);
}

static inline uint8_t* pktmbuf_mtod_wrapper(struct rte_mbuf* pkt){
	return rte_pktmbuf_mtod(pkt,uint8_t*);
}
extern int onvm_init(struct onvm_nf_local_ctx **);
extern void onvm_send_pkt(char *, int, struct onvm_nf_local_ctx *,int);
*/
import "C"

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"unsafe"
)

var pktmbuf_pool *C.struct_rte_mempool
var pktCount int
var config = &Config{} //move config to global
var nf_ctx *C.struct_onvm_nf_local_ctx
var channelMap = make(map[ConnMeta]chan PktMeta) //map to hanndle each channel of connection

//for each connection
type ConnMeta struct {
	ip       string
	port     int
	protocol int
}

type PktMeta struct {
	srcIp      net.IP
	srcPort    int
	payloadLen int     //packet length just include payload after tcpudp
	payloadPtr *[]byte //the pointer to byte slice of payload
}

type Config struct {
	//ServiceID int `yaml:"serviceID"`
	IPIDMap []struct {
		IP *string `yaml:"IP"`
		ID *int32  `yaml:"ID"`
	} `yaml:"IPIDMap"`
}

type OnvmConn struct {
	laddr   *net.UDPAddr
	udpChan chan PktMeta
}

func init() {
	C.onvm_init(&nf_ctx)
}

//export Handler
func Handler(pkt *C.struct_rte_mbuf, meta *C.struct_onvm_pkt_meta,
	nf_local_ctx *C.struct_onvm_nf_local_ctx) int32 {
	pktCount++
	recvLen := int(C.pktmbuf_data_len_wrapper(pkt))                               //length include header??//int(C.rte_pktmbuf_data_len(pkt))
	buf := C.GoBytes(unsafe.Pointer(C.pktmbuf_mtod_wrapper(pkt)), C.int(recvLen)) //turn c memory to go memory
	umsBuf, raddr := unMarshalUDP(buf)
	udpMeta := ConnMeta{
		raddr.IP.String(),
		raddr.Port,
		17,
	}
	pktMeta := PktMeta{
		raddr.IP,
		raddr.Port,
		len(umsBuf),
		&umsBuf,
	}
	channel, ok := channelMap[udpMeta]
	if ok {
		channel <- pktMeta
	} else {
		//drop packet(?)
	}

	meta.action = C.ONVM_NF_ACTION_DROP

	return 0
}

//to regist channel and it connection meta to map
func (conn *OnvmConn) registerChannel() {
	udpTuple := ConnMeta{
		conn.laddr.IP.String(),
		conn.laddr.Port,
		17,
	}
	conn.udpChan = make(chan PktMeta, 1)
	channelMap[udpTuple] = conn.udpChan
}

func ListenUDP(network string, laddr *net.UDPAddr) (*OnvmConn, error) {
	// Read Config
	var ipIdConfig string
	if dir, err := os.Getwd(); err != nil {
		ipIdConfig = "./ipid.yaml"
	} else {
		ipIdConfig = dir + "/ipid.yaml"
	}
	if os.Getenv("IPIDConfig") != "" {
		ipIdConfig = os.Getenv("IPIDConfig")
	}
	fmt.Printf("Read config from %s", ipIdConfig)
	if yamlFile, err := ioutil.ReadFile(ipIdConfig); err != nil {
		panic(err)
	} else {
		if unMarshalErr := yaml.Unmarshal(yamlFile, config); unMarshalErr != nil {
			panic(unMarshalErr)
		}
	}

	conn := &OnvmConn{}
	//store local addr
	conn.laddr = laddr
	//register
	conn.registerChannel()

	pktmbuf_pool = C.rte_mempool_lookup(C.CString("MProc_pktmbuf_pool"))
	if pktmbuf_pool == nil {
		return nil, fmt.Errorf("pkt alloc from pool failed")
	}

	go C.onvm_nflib_run(nf_ctx)

	return conn, nil
}

func (conn *OnvmConn) LocalAddr() net.Addr {
	laddr := conn.laddr
	return laddr
}

func (conn *OnvmConn) Close() {

	C.onvm_nflib_stop(nf_ctx)
	//deregister channel
	udpMeta := &ConnMeta{
		conn.laddr.IP.String(),
		conn.laddr.Port,
		17,
	}
	delete(channelMap, *udpMeta) //delete from map
	fmt.Println("Close onvm UDP")
}

func (conn *OnvmConn) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	var success_send_len int
	var buffer_ptr *C.char //point to the head of byte data
	//look up table to get id
	ID, err := ipToID(addr.IP)
	if err != nil {
		return 0, err
	}
	success_send_len = len(b) //???ONVM has functon to get it?-->right now onvm_send_pkt return void
	tempBuffer := marshalUDP(b, addr, conn.laddr)
	buffer_ptr = getCPtrOfByteData(tempBuffer)
	C.onvm_send_pkt(buffer_ptr, C.int(ID), nf_ctx, C.int(len(tempBuffer))) //C.onvm_send_pkt havn't write?

	return success_send_len, err
}

func (conn *OnvmConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	var pktMeta PktMeta
	pktMeta = <-conn.udpChan
	recvLength := pktMeta.payloadLen
	copy(b, *(pktMeta.payloadPtr))
	raddr := &net.UDPAddr{
		IP:   pktMeta.srcIp,
		Port: pktMeta.srcPort,
	}
	return recvLength, raddr, nil

}

func ipToID(ip net.IP) (Id int, err error) {
	Id = -1
	for i := range config.IPIDMap {
		if *config.IPIDMap[i].IP == ip.String() {
			Id = int(*config.IPIDMap[i].ID)
			break
		}
	}
	if Id == -1 {
		err = fmt.Errorf("no match id")
	}
	return
}

func marshalUDP(b []byte, raddr *net.UDPAddr, laddr *net.UDPAddr) []byte {
	//interfacebyname may need to modify,not en0
	/*
		ifi ,err :=net.InterfaceByName("en0")
		if err!=nil {
			panic(err)
		}
	*/
	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	ethlayer := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0, 0, 0, 0, 0, 0},
		DstMAC:       net.HardwareAddr{0, 0, 0, 0, 0, 0},
		EthernetType: layers.EthernetTypeIPv4,
	}

	iplayer := &layers.IPv4{
		Version:  uint8(4),
		SrcIP:    laddr.IP,
		DstIP:    raddr.IP,
		TTL:      64,
		Protocol: layers.IPProtocolUDP,
	}

	udplayer := &layers.UDP{
		SrcPort: layers.UDPPort(laddr.Port),
		DstPort: layers.UDPPort(raddr.Port),
	}
	udplayer.SetNetworkLayerForChecksum(iplayer)
	err := gopacket.SerializeLayers(buffer, options,
		ethlayer,
		iplayer,
		udplayer,
		gopacket.Payload(b),
	)
	if err != nil {
		panic(err)
	}
	outgoingpacket := buffer.Bytes()
	return outgoingpacket

}

func getCPtrOfByteData(b []byte) *C.char {
	shdr := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	ptr := (*C.char)(unsafe.Pointer(shdr.Data))
	//runtime alive?
	return ptr
}

func unMarshalUDP(input []byte) (payLoad []byte, rAddr *net.UDPAddr) {
	//Unmarshaludp header and get the information(ip port) from header
	var rPort int
	var rIp net.IP
	ethPacket := gopacket.NewPacket(
		input,
		layers.LayerTypeEthernet,
		gopacket.NoCopy) //this may be type zero copy

	ipLayer := ethPacket.Layer(layers.LayerTypeIPv4)

	if ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		rIp = ip.SrcIP
	}
	udpLayer := ethPacket.Layer(layers.LayerTypeUDP)
	if udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		rPort = int(udp.SrcPort)
		payLoad = udp.Payload
	}

	rAddr = &net.UDPAddr{
		IP:   rIp,
		Port: rPort,
	}

	return
}
