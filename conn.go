package goConn

// #cgo CFLAGS: -I/home/ubuntu/openNetVM/onvm/onvm_nflib
// #cgo CFLAGS: -I/home/ubuntu/openNetVM/onvm/lib
// #cgo CFLAGS: -I/home/ubuntu/openNetVM/dpdk/x86_64-native-linuxapp-gcc/include
// #cgo LDFLAGS: /home/ubuntu/openNetVM/onvm/onvm_nflib/x86_64-native-linuxapp-gcc/libonvm.a
// #cgo LDFLAGS: /home/ubuntu/openNetVM/onvm/lib/x86_64-native-linuxapp-gcc/lib/libonvmhelper.a -lm
/*
extern int onvmInit();
*/
import "C"
import (
    "fmt"
)

var udpChan = make(chan * C.struct_rte_mbuf, 1)
var pktCount int

type OnvmConn struct {
    nf_ctx * C.struct_onvm_nf_local_ctx
    udpChan chan * C.struct_rte_mbuf
}

//export Handler
func Handler(pkt * C.struct_rte_mbuf, meta * C.struct_onvm_pkt_meta,
                    nf_local_ctx * C.struct_onvm_nf_local_ctx) int {
    pktCount++
    fmt.Println("packet received!")
    meta.action = C.ONVM_NF_ACTION_DROP

    udp_hdr := C.get_pkt_udp_hdr(pkt);

    if udp_hdr.dst_port == 2125 {
        udpChan <- pkt
    }
    return 0;
}

//export ListenUDP
func ListenUDP(network string,laddr *net.UDPAddr) {
    conn := &OnvmConn {
    }

    C.onvmInit(conn.nf_ctx)

    pktmbuf_pool = C.rte_mempool_lookup(C.CString("MProc_pktmbuf_pool"));
    if (pktmbuf_pool == nil) {
        return -1
    }

    go conn.udpHandler()
    go C.onvm_nflib_run(conn.nf_ctx);

    fmt.Printf("ListenUDP: %s\n", network)
    return 0;
}

//export Close
func (conn * OnvmConn)Close() {

    C.onvm_nflib_stop(conn.nf_ctx)

    fmt.Println("Close onvm UDP")
}

func (conn * OnvmConn)udpHandler() {
    for {
        select {
            case <- udpChan:
              fmt.Println("Receive UDP")
        }
    }
}
