#!/bin/zsh

#tee in_pylsp.log | pylsp | tee out_pylsp.log
tee in_pylsp.log | pyright-langserver --stdio | tee out_pylsp.log
