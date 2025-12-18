#!/bin/zsh

export GOPATH="$HOME/go"
PATH="${PATH}:$GOPATH/bin"

tee in_gopls.log | gopls | tee out_gopls.log
