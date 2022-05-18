#!/bin/bash

# Start a node with default as ropsten chain.
export NETWORK=${NETWORK:-testnet}

if [ "${NETWORK}" == "mainnet" ] || [ "${NETWORK}" == "MAINNET" ]; then
    exec /app/geth --config=/app/geth.toml --gcmode=archive
else
    exec /app/geth --ropsten --config=/app/geth.toml --gcmode=archive
fi