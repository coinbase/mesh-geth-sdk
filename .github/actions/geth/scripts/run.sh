#!/bin/sh

set -eu

geth \
    --verbosity 5 \
    --nodiscover \
    --syncmode 'full' \
    --nat none \
    --port 30310 \
    --http \
    --http.addr '0.0.0.0' \
    --http.port 8545 \
    --http.vhosts '*' \
    --http.api 'personal,eth,net,web3,txpool,debug' \
    --networkid '1337' \
    --unlock "0,1,2" \
    --password /root/config/password.txt \
    --allow-insecure-unlock

