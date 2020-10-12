package onvmNet

// #cgo CFLAGS: -I/home/ubuntu/openNetVM/onvm/onvm_nflib
// #cgo CFLAGS: -I/home/ubuntu/openNetVM/onvm/lib
// #cgo CFLAGS: -I/home/ubuntu/openNetVM/dpdk/x86_64-native-linuxapp-gcc/include
// #cgo LDFLAGS: /home/ubuntu/openNetVM/onvm/onvm_nflib/x86_64-native-linuxapp-gcc/libonvm.a
// #cgo LDFLAGS: /home/ubuntu/openNetVM/onvm/lib/x86_64-native-linuxapp-gcc/lib/libonvmhelper.a -lm
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
*/
import "C"

import (
    "fmt"
    "net"
    "io/ioutil"
    "unsafe"

    "gopkg.in/yaml.v2"
)

var udpChan = make(chan * C.struct_rte_mbuf, 1)
var handToReadChan = make(chan * C.struct_rte_mbuf,1)

var pktmbuf_pool * C.struct_rte_mempool
var pktCount int

type Config struct {
    ServiceID int32 `yaml:"serviceID"`
}

type OnvmConn struct {
    nf_ctx * C.struct_onvm_nf_local_ctx
    udpChan chan * C.struct_rte_mbuf
    handToReadChan chan * C.struct_rte_mbuf
}

//export Handler
func Handler(pkt * C.struct_rte_mbuf, meta * C.struct_onvm_pkt_meta,
                    nf_local_ctx * C.struct_onvm_nf_local_ctx) int32 {
    pktCount++
    fmt.Println("packet received!")
    meta.action = C.ONVM_NF_ACTION_DROP

    udp_hdr := C.get_pkt_udp_hdr(pkt);

    if udp_hdr.dst_port == 2125 {
        udpChan <- pkt
    }
    return 0;
}

func ListenUDP(network string, laddr *net.UDPAddr) (*OnvmConn, error) {
    // Read Config
    var config Config
    if yamlFile, err := ioutil.ReadFile("udp.yaml"); err != nil {
        panic(err)
    } else {
        yaml.Unmarshal(yamlFile, config)
    }

    conn := &OnvmConn {
    }

    C.onvmInit(conn.nf_ctx, C.int(config.ServiceID))

    pktmbuf_pool = C.rte_mempool_lookup(C.CString("MProc_pktmbuf_pool"));
    if (pktmbuf_pool == nil) {
        return nil, fmt.Errorf("pkt alloc from pool failed")
    }

    go conn.udpHandler()
    go C.onvm_nflib_run(conn.nf_ctx);

    fmt.Printf("ListenUDP: %s\n", network)
    return conn, nil;
}

func (conn * OnvmConn)Close() {

    C.onvm_nflib_stop(conn.nf_ctx)

    fmt.Println("Close onvm UDP")
}

func (conn * OnvmConn)udpHandler() {
    var relayBuf * C.struct_rte_mbuf
    for {
        select {
            case relayBuf = <- udpChan:
              fmt.Println("Receive UDP")
              handToReadChan <- relayBuf//send it to readfromudp
        }
    }
}

func (conn * OnvmConn) WriteToUDP(b []byte ,addr * net.UDPAddr)(int,error){
    var addr C.struct_sockaddr_in
    var success_send_len int
    success_send_len = 0//???ONVM has functon to get it?
    tempbuffer:=marshalUDP(b,addr)//haven't done
    //send the message to where???????
    C.ONVMSEND(&tempbuffer[0],conn.nf_ctx)

    return success_send_len,nil
}

func (conn * OnvmConn) ReadFromUDP(b []byte)(int,*net.UDPAddr,error){
    buf := make([]byte,1500)
    var buffer_ptr *C.char
    buffer_ptr = C.CString(buf)
    var onvm_addr * C.struct_rte_mbuf
    onvm_addr = <-conn.handToReadChan
    var recv_length = 0 //????????onvm has function to get the length of buffer
    C.memcpy(unsafe.Pointer(buffer_ptr),unsafe.Pointer(onvm_addr),recv_length)//??length not sure
    //C.memcpy(unsafe.Pointer(&b[0]),unsafe.Pointer(onvm_addr),1500)//??length not sure
    buf = C.GoString(buffer_ptr)
    raddr := unMarshalUDP()

    return recv_length,raddr,nil

}
func marshalUDP(b []byte,addr *net.UDPAddr)(output []byte){
    //wrapper payload with layer2 and layer3
    return
}
func unMarshalUDP(input []byte,output []byte)(*net.UDPAddr){
    //Unmarshaludp header and get the information(ip port) from header
    return nil
}

