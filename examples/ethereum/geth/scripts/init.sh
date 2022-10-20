#!/bin/sh

set -eu

geth init /root/config/genesis.json
geth --password /root/config/password.txt account import /root/config/private-key.txt 
