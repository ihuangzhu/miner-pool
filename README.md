# miner-pool
Open source eth miner pool

The mining pool refers to the [open-ethereum-pool](https://github.com/sammy007/open-ethereum-pool). Currently, only the basic mining functions are implemented, and the recording of proof of work and settlement have not yet been implemented.

矿池参考 [open-ethereum-pool](https://github.com/sammy007/open-ethereum-pool) 开发，目前只实现了基础的挖矿功能，记录工作证明及结算还没有实现。

> #### Work Process: 
> - geth(getWork or notify) -> miner-pool(broadcast work) -> miner machine
> - miner-machine(mining PoW) -> miner-pool(check and post work then record) -> geth(submitWork and spread) 

`config.json`
```json 
{
  "name": "main",

  "proxy": {
    "enabled": true,
    "listen": "0.0.0.0:8008",
    "timeout": "120s",
    "maxConn": 1024,
    "target": "0x000000007fffffffffffffffffffffffffffffffffffffffffffffffffffffff",

    "daemon": {
      "hosts": "127.0.0.1",
      "port": 8545,
      "notifyWorkUrl": "0.0.0.0:8107"
    }
  }
}
```
