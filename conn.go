package onvmNet

// #cgo CFLAGS: -m64 -pthread -O3 -march=native
// #cgo CFLAGS: -I/root/openNetVM/onvm/onvm_nflib
// #cgo CFLAGS: -I/root/openNetVM/onvm/lib
// #cgo CFLAGS: -I/root/openNetVM/dpdk/x86_64-native-linuxapp-gcc/include
// #cgo LDFLAGS: /root/openNetVM/onvm/onvm_nflib/x86_64-native-linuxapp-gcc/libonvm.a
// #cgo LDFLAGS: /root/openNetVM/onvm/lib/x86_64-native-linuxapp-gcc/lib/libonvmhelper.a -lm
// #cgo LDFLAGS: -L/root/openNetVM/dpdk/x86_64-native-linuxapp-gcc/lib
// #cgo LDFLAGS: -lrte_flow_classify -Wl,--whole-archive -lrte_pipeline -Wl,--no-whole-archive -Wl,--whole-archive -lrte_table -Wl,--no-whole-archive -Wl,--whole-archive -lrte_port -Wl,--no-whole-archive -lrte_pdump -lrte_distributor -lrte_ip_frag -lrte_meter -lrte_lpm -Wl,--whole-archive -lrte_acl -Wl,--no-whole-archive -lrte_jobstats -lrte_metrics -lrte_bitratestats -lrte_latencystats -lrte_power -lrte_efd -lrte_bpf -Wl,--whole-archive -lrte_cfgfile -lrte_gro -lrte_gso -lrte_hash -lrte_member -lrte_vhost -lrte_kvargs -lrte_mbuf -lrte_net -lrte_ethdev -lrte_bbdev -lrte_cryptodev -lrte_security -lrte_compressdev -lrte_eventdev -lrte_rawdev -lrte_timer -lrte_mempool -lrte_mempool_ring -lrte_ring -lrte_pci -lrte_eal -lrte_cmdline -lrte_reorder -lrte_sched -lrte_kni -lrte_common_cpt -lrte_common_octeontx -lrte_common_dpaax -lrte_bus_pci -lrte_bus_vdev -lrte_bus_dpaa -lrte_bus_fslmc -lrte_mempool_bucket -lrte_mempool_stack -lrte_mempool_dpaa -lrte_mempool_dpaa2 -lrte_pmd_af_packet -lrte_pmd_ark -lrte_pmd_atlantic -lrte_pmd_avf -lrte_pmd_avp -lrte_pmd_axgbe -lrte_pmd_bnxt -lrte_pmd_bond -lrte_pmd_cxgbe -lrte_pmd_dpaa -lrte_pmd_dpaa2 -lrte_pmd_e1000 -lrte_pmd_ena -lrte_pmd_enetc -lrte_pmd_enic -lrte_pmd_fm10k -lrte_pmd_failsafe -lrte_pmd_i40e -lrte_pmd_ixgbe -lrte_pmd_kni -lrte_pmd_lio -lrte_pmd_nfp -lrte_pmd_null -lrte_pmd_qede -lrte_pmd_ring -lrte_pmd_softnic -lrte_pmd_sfc_efx -lrte_pmd_tap -lrte_pmd_thunderx_nicvf -lrte_pmd_vdev_netvsc -lrte_pmd_virtio -lrte_pmd_vhost -lrte_pmd_ifc -lrte_pmd_vmxnet3_uio -lrte_bus_vmbus -lrte_pmd_netvsc -lrte_pmd_bbdev_null -lrte_pmd_null_crypto -lrte_pmd_octeontx_crypto -lrte_pmd_crypto_scheduler -lrte_pmd_dpaa2_sec -lrte_pmd_dpaa_sec -lrte_pmd_caam_jr -lrte_pmd_virtio_crypto -lrte_pmd_octeontx_zip -lrte_pmd_qat -lrte_pmd_skeleton_event -lrte_pmd_sw_event -lrte_pmd_dsw_event -lrte_pmd_octeontx_ssovf -lrte_pmd_dpaa_event -lrte_pmd_dpaa2_event -lrte_mempool_octeontx -lrte_pmd_octeontx -lrte_pmd_opdl_event -lrte_pmd_skeleton_rawdev -lrte_pmd_dpaa2_cmdif -lrte_pmd_dpaa2_qdma -lrte_bus_ifpga -lrte_pmd_ifpga_rawdev -Wl,--no-whole-archive -lrt -lm -lnuma -ldl
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
extern int onvm_init(struct onvm_nf_local_ctx *, int);
extern void onvm_send_pkt(char *, int, struct onvm_nf_local_ctx *);
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
	"os"
)

var udpChan = make(chan EthFrame, 1)
var handToReadChan = make(chan EthFrame,1)
var pktmbuf_pool *C.struct_rte_mempool
var pktCount int
var config = &Config{} //move config to global

type EthFrame struct {
	frame     *C.struct_rte_mbuf
	frame_len int
}

type Config struct {
	ServiceID int `yaml:"serviceID"`
	IPIDMap   []struct {
		IP *string `yaml:"IP"`
		ID *int32  `yaml:"ID"`
	} `yaml:"IPIDMap"`
}

type OnvmConn struct {
	laddr *net.UDPAddr
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
		//udpChan <- EthFrame { pkt, int(C.rte_pktmbuf_data_len(pkt)) }
		udpChan <- EthFrame{pkt, 5}
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
	dir, _ := os.Getwd()
	fmt.Printf("Read config from %s/onvmNet/udp.yaml", dir)
	//config := &Config{}//move to global
	if yamlFile, err := ioutil.ReadFile("./onvmNet/udp.yaml"); err != nil {
		panic(err)
	} else {
		if unMarshalErr := yaml.Unmarshal(yamlFile, config); unMarshalErr != nil {
			panic(unMarshalErr)
		}
	}

	conn := &OnvmConn{}
	//store local addr
	conn.laddr = laddr

	C.onvm_init(conn.nf_ctx, C.int(config.ServiceID))
	//C.onvmInit(conn.nf_ctx, C.int(1))

	pktmbuf_pool = C.rte_mempool_lookup(C.CString("MProc_pktmbuf_pool"))
	if pktmbuf_pool == nil {
		return nil, fmt.Errorf("pkt alloc from pool failed")
	}

	go conn.udpHandler()
	go C.onvm_nflib_run(conn.nf_ctx)

	fmt.Printf("ListenUDP: %s\n", network)
	return conn, nil
}

//func (conn * OnvmConn) LocalAddr() (laddr net.Addr) {
//}

func (conn *OnvmConn) Close() {

	C.onvm_nflib_stop(conn.nf_ctx)

	fmt.Println("Close onvm UDP")
}

func (conn * OnvmConn) WriteToUDP(b []byte ,addr * net.UDPAddr)(int,error){
    var success_send_len int
    var buffer_ptr *C.char //point to the head of byte data
    var ID int
	//look up table to get id 
    ID = ipToID(addr.IP)
    success_send_len = 0//???ONVM has functon to get it?-->right now onvm_send_pkt return void
    tempBuffer:= marshalUDP(b,addr,conn.laddr)
    buffer_ptr = getCPtrOfByteData(tempBuffer)
	C.onvm_send_pkt(buffer_ptr,C.int(ID),conn.nf_ctx)//C.onvm_send_pkt havn't write?

    return success_send_len,nil
}

func (conn * OnvmConn) ReadFromUDP(b []byte)(int,*net.UDPAddr,error){
	var ethFame EthFrame
    ethFame = <- handToReadChan
    recvLength := ethFame.frame_len
	header_len := 80//need to get whole length include layer23
    buf := C.GoBytes(unsafe.Pointer(ethFame.frame),C.int(recvLength+header_len))
    umsBuf,raddr := unMarshalUDP(buf)
    copy(b,umsBuf)
    return recvLength,raddr,nil

}

func ipToID(ip net.IP)(Id int){
	for i := range config.IPIDMap{
		if *config.IPIDMap[i].IP == ip.String(){
			Id = int(*config.IPIDMap[i].ID)
			break
		}
	}
	return
}

func marshalUDP(b []byte,raddr *net.UDPAddr,laddr *net.UDPAddr)([]byte){
	//interfacebyname may need to modify,not en0
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
		SrcIP: laddr.IP,
		DstIP: raddr.IP,
		TTL: 64,
		Protocol: layers.IPProtocolUDP,
	}

	udplayer := &layers.UDP{
		SrcPort: layers.UDPPort(laddr.Port),
		DstPort: layers.UDPPort(raddr.Port),
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
		gopacket.NoCopy)//this may be type zero copy

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


