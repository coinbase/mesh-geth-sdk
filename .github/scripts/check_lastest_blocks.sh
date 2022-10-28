#!/bin/bash

# downloading cli
curl -sSfL https://raw.githubusercontent.com/coinbase/rosetta-cli/master/scripts/install.sh | sh -s

block_tip=($(curl -s --location --request POST 'http://localhost:8080/network/status' \
--header 'Content-Type: application/json' \
--data-raw '{
    "network_identifier": {
        "blockchain": "Ethereum",
        "network": "Mainnet"
    }
}' | python3 -c 'import json,sys;obj=json.load(sys.stdin);print(obj["current_block_identifier"]["index"])'))

lastest_X_blocks=10
start_index=$(($block_tip - $lastest_X_blocks))

echo "start check:data at " $start_index
./bin/rosetta-cli --configuration-file examples/ethereum/rosetta-cli-conf/mainnet/config.json check:data --start-block $start_index
