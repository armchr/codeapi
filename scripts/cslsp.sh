#!/bin/zsh

# https://github.com/OmniSharp/omnisharp-roslyn
# https://github.com/razzmatazz/csharp-language-server
#
#tee in_cslsp.log | OmniSharp --stdio | tee out_cslsp.log
tee in_cslsp.log | csharp-ls 2>&1 | tee out_cslsp.log
