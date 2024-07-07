#!/bin/bash

clear
# Assuming GNU screen is installed and folder of astra executable is added to $PATH
# To do this, add "setenv PATH /path/to/astra:$PATH" to /etc/screenrc
sudo screen -S test_astra -d -m astra -p 8000 -c /tmp/astra.conf
read -p "Press enter to stop astra"
sudo screen -S test_astra -X at 0 stuff ^C
