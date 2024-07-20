#!/bin/bash

clear
# Assuming GNU screen is installed and folder of astra executable is added to $PATH
# To do this, add "setenv PATH /path/to/astra:$PATH" to /etc/screenrc
sudo screen -S test_analyzer -d -m astra --analyze -p 8001
read -p "Press enter to stop analyzer"
sudo screen -S test_analyzer -X at 0 stuff ^C
