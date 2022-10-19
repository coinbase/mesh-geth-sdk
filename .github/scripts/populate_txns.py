from web3 import Web3
from web3.middleware import geth_poa_middleware
from eth_account import Account

web3 = Web3(Web3.HTTPProvider("http://127.0.0.1:8546"))
web3.middleware_onion.inject(geth_poa_middleware, layer=0)

print("latest block", web3.eth.block_number)

dst1 = Account.create('KEYSMASH FJAFJKLDSKF7JKFDJ 2121')
dst2 = Account.create('KEYSMASH dassaad 441')


for _ in range(5):
    web3.eth.send_transaction({
        'to': dst2.address,
        'from': web3.eth.coinbase,
        'value': 12345,
        'gas': 21000,
        'maxFeePerGas': web3.toWei(250, 'gwei'),
        'maxPriorityFeePerGas': web3.toWei(2, 'gwei'),
    })

    web3.eth.send_transaction({
        'to': dst1.address,
        'from': web3.eth.coinbase,
        'value': 100000,
        'gas': 21000,
        'maxFeePerGas': web3.toWei(250, 'gwei'),
        'maxPriorityFeePerGas': web3.toWei(2, 'gwei'),
    })

print("latest block", web3.eth.block_number)
