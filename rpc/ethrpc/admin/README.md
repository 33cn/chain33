# admin_peers
>实现查询peer 节点

req

````
{
  "method":"admin_peers",
  "params":[],
  "id":1,
  "jsonrpc":"2.0"
}
````

response
````
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
        {
            "addr": "192.168.0.19",
            "port": 13803,
            "name": "16Uiu2HAmBdwm5i6Ao6hBedNXHSM44ZUhM4243s5yJGAKPyHRjESw",
            "mempoolSize": 0,
            "self": true,
            "header": {
                "version": 0,
                "parentHash": "0x56a00b884045e7087a6df93b58da8ba06d8127e3f9bf822b3c57271bf493aa62",
                "txHash": "0x91b60784d75497bc7c2ed681e51a985d933ec63ee6adf032ba6024a6adcb3dc9",
                "stateHash": "0x593cfe7c38f21d87f2377983a911601140b679763e7e0439b830eab45dc26187",
                "height": 2,
                "blockTime": 1647411454,
                "txCount": 1,
                "hash": "0x560f19205f79a41bd89ccfc288dc89fe9902c33afafce9a43d00ed5c6c8e082e",
                "difficulty": 0
            },
            "version": "6.0.0-3877-g1405d606a-1405d606@1.0.0",
            "runningTime": "0.044 minutes"
        }
    ]
}
````


# admin_nodeInfo
>查询节点信息

req
````
{
  "method":"admin_nodeInfo",
  "params":[],
  "id":1,
  "jsonrpc":"2.0"
}
````

response 
````
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "addr": "192.168.0.19",
        "port": 13803,
        "name": "16Uiu2HAmBdwm5i6Ao6hBedNXHSM44ZUhM4243s5yJGAKPyHRjESw",
        "mempoolSize": 0,
        "self": true,
        "header": {
            "version": 0,
            "parentHash": "0x56a00b884045e7087a6df93b58da8ba06d8127e3f9bf822b3c57271bf493aa62",
            "txHash": "0x91b60784d75497bc7c2ed681e51a985d933ec63ee6adf032ba6024a6adcb3dc9",
            "stateHash": "0x593cfe7c38f21d87f2377983a911601140b679763e7e0439b830eab45dc26187",
            "height": 2,
            "blockTime": 1647411454,
            "txCount": 1,
            "hash": "0x560f19205f79a41bd89ccfc288dc89fe9902c33afafce9a43d00ed5c6c8e082e",
            "difficulty": 0
        },
        "version": "6.0.0-3877-g1405d606a-1405d606@1.0.0",
        "runningTime": "0.262 minutes"
    }
}
````

# admin_datadir
>查询数据存储目录

req
````
req
{
  "method":"admin_datadir",
  "params":[],
  "id":1,
  "jsonrpc":"2.0"
}
````

response
````
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "datadir"
}
````