#!/bin/bash

clear
# Assuming GNU screen is installed and folder of astra executable is added to $PATH
# To do this, add "setenv PATH /path/to/astra:$PATH" to /etc/screenrc
sudo screen -S astra_test_check -d -m astra --analyze -p 8001
go clean -testcache
go test -v -timeout 10m -race -run ^TestCheck$ m3u_merge_astra/astra/analyzer
sudo screen -X -S astra_test_check kill
