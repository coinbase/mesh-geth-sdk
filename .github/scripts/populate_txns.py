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
        'to': '0xd3CdA913deB6f67967B99D67aCDFa1712C293601',
        'from': web3.eth.coinbase,
        'value': 12345
    })

print("latest block", web3.eth.block_number)
