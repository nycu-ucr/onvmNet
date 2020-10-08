# onvmNet

```bash
export CGO_CFLAGS="-m64 -pthread -O3 -march=native -I${RTE_SDK}/${RTE_TARGET}/include -I${ONVM_HOME}/onvm/onvm_nflib/ -I${ONVM_HOME}/onvm/lib/"
export CGO_LDFLAGS="`cat $RTE_SDK/$RTE_TARGET/lib/ldflags.txt` -L${ONVM_HOME}/onvm/onvm_nflib/${RTE_TARGET}/lib -lm"
```
