#!/bin/bash

nohup make run-rosetta > /dev/null 2>&1 &

sleep 180

block_tip=($(curl -s --location --request POST 'http://localhost:8000/network/status' \
--header 'Content-Type: application/json' \
--data-raw '{
    "network_identifier": {
        "blockchain": "Ethereum",
        "network": "Mainnet"
    }
}' | python3 -c 'import json,sys;obj=json.load(sys.stdin);print(obj["sync_status"]["target_index"])'))

echo $(seq $START_INDEX $BLKS_PER_CHUNK $block_tip | jq -cnR '[inputs | select(length>0)]')
