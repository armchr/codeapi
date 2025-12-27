#!/bin/zsh

# https://github.com/eclipse-jdtls/eclipse.jdt.ls

export PATH="${CODEAPI_ROOT}/assets/bin/:${PATH}"
tee in_javalsp.log | jdtls | tee out_javalsp.log
