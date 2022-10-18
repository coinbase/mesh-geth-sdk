#!/bin/bash

END_INDEX=$(($START_INDEX+$COUNT))
echo "index ranges", $START_INDEX, $END_INDEX

cd examples/ethereum
nohup make run-rosetta > /dev/null 2>&1 &

# downloading cli
curl -sSfL https://raw.githubusercontent.com/coinbase/rosetta-cli/master/scripts/install.sh | sh -s

sleep 180

echo "start check:data"
./bin/rosetta-cli --configuration-file examples/ethereum/rosetta-cli-conf/mainnet/config.json check:data --start-block $START_INDEX --end-block $END_INDEX
