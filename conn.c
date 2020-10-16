#include <stdio.h>
#include "_cgo_export.h"

#define NF_TAG "go-nf"
#define SERVICE_ID_DIGIT 10

//typedef int GoInt;
//extern int Handler(struct rte_mbuf*, struct onvm_pkt_meta*, struct onvm_nf_local_ctx*);

int onvm_init(struct onvm_nf_local_ctx *nf_local_ctx, int serviceId) {
    int arg_offset;
    struct onvm_nf_function_table *nf_function_table;

    nf_local_ctx = onvm_nflib_init_nf_local_ctx();
    onvm_nflib_start_signal_handler(nf_local_ctx, NULL);

    nf_function_table = onvm_nflib_init_nf_function_table();
    nf_function_table->pkt_handler = &Handler;

    char service_id_str[SERVICE_ID_DIGIT];
    sprintf(service_id_str, "%d", serviceId);
    char * cmd[2] = {"./go.sh", service_id_str};
    if ((arg_offset = onvm_nflib_init(2, cmd, NF_TAG, nf_local_ctx, nf_function_table)) < 0) {
            onvm_nflib_stop(nf_local_ctx);
            if (arg_offset == ONVM_SIGNAL_TERMINATION) {
                    printf("Exiting due to user termination\n");
                    return 0;
            } else {
                    rte_exit(EXIT_FAILURE, "Failed ONVM init\n");
            }
    }

    return 0;
}

void onvm_send_pkt(char * buff, int service_id, struct onvm_nf_local_ctx * ctx) {
}

