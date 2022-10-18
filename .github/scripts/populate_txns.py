from web3 import Web3
from web3.middleware import geth_poa_middleware

web3 = Web3(Web3.HTTPProvider("http://127.0.0.1:8546"))
web3.middleware_onion.inject(geth_poa_middleware, layer=0)

print("latest block", web3.eth.block_number)

# transfer 12345 to account 0
web3.eth.send_transaction({
  'to': web3.eth.accounts[0],
  'from': web3.eth.coinbase,
  'value': 12345
})

print("latest block", web3.eth.block_number)
