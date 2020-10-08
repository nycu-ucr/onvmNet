#include <stdio.h>
#include "_cgo_export.h"

#define NF_TAG "go-nf"

//typedef int GoInt;

int onvmInit(struct onvm_nf_local_ctx *nf_local_ctx) {
    int arg_offset;
    struct onvm_nf_function_table *nf_function_table;

    nf_local_ctx = onvm_nflib_init_nf_local_ctx();
    onvm_nflib_start_signal_handler(nf_local_ctx, NULL);

    nf_function_table = onvm_nflib_init_nf_function_table();
    nf_function_table->pkt_handler = &Handler;

    if ((arg_offset = onvm_nflib_init(argc, argv, NF_TAG, nf_local_ctx, nf_function_table)) < 0) {
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
