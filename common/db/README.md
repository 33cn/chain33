# 数据库配置选项说明

目前支持 [leveldb](https://github.com/syndtr/goleveldb) 和 [badger](https://github.com/dgraph-io/badger)（测试阶段）
两种可选数据库做为KV数据存储，默认为leveldb  
参考性能测试：https://github.com/suyanlong/dbcompare 总结出： 
- 优点：   
  - badger在机械硬盘环境下，读写速度和leveldb相差不大，但在SSD固态硬盘环境下，读写速度都快于leveldb
  - badger的KV分离存储，所以相同数据情况下LSM会更小，可以全部加载到RAM，提高读写速度
- 缺点：  
  - badger的value是存储在valuelog中的，默认超过1KB的value才会压缩存储，会导致valuelog有很多冗余信息（可通过DB.RunValueLogGC()进行回收），占用磁盘空间会比leveldb多

## leveldb
选用 [leveldb](https://github.com/syndtr/goleveldb) 做为KV数据存储  
修改chain33.toml文件中，[blockchain]、[store]、[localstore]、[wallet] 标签中driver的值为leveldb

```toml
{
    "driver": "leveldb"
}
```

## badger
选用 [badger](https://github.com/dgraph-io/badger) 做为KV数据存储  
修改chain33.toml文件中，[blockchain]、[store]、[localstore]、[wallet] 标签中driver的值为gobadgerdb
```toml
{
    "driver": "gobadgerdb"
}
```

# 实现自定义数据库接口说明


