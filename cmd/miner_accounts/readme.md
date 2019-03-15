## 挖矿监控

数据获得:
 1. 现在默认配置了8个挖矿帐号， 不传递帐号， 会返回10个挖矿帐号的相关信息。
 1. 但以后挖矿帐号可能变化， 可以通过修改服务的配置， 也可以直接在监控端修改请求。直接把监控的地址列表放在请求里
```
wget 127.0.0.1:8866 --no-proxy --post-data='{"id" : 1 , "method" : "ShowMinerAccount.Get", "params" :  [{ }] } '
wget 127.0.0.1:8866 --no-proxy --post-data='{"id" : 1 , "method" : "ShowMinerAccount.Get", "params" :  [{ "addrs" : ["1PSYYfCbtSeT1vJTvSKmQvhz8y6VhtddWi"] }] } '
```

结果：
 1. seconds 里是实际挖矿时间。 理论上返回是一小时， 但实际上可能是有误差的
 2. totalIncreate 这段时间挖矿的所有币
 3. minerAccounts 一共配置里的10个挖矿帐号信息
 4. increase 是对应挖矿帐号的这段时间挖矿所得
 5. total 是对应挖矿帐号里的所有的币
 6. addr 是对应挖矿帐号的地址
 7. frozen 冻结挖矿的币
 8. expectIncrease 预期挖到多少币
 9. expectMinerBlocks 预期间隔多少个块能挖到币
 10. minerBtyDuring 在预期能挖到币的两倍时间间隔内，挖到多少币
 11. expectTotalIncrease  一个小时内预期挖矿增加

需要监控
 1. 挖矿总量异常： 根据总体币的挖矿情况报警 （不再根据原来的固定值）
    1. 随着挖矿的bty变动， 用于挖矿的币会有变化， 按固定的值来预警， 在用于挖矿的币变少时会有误报， 在json中增加 expectTotalIncrease 项， 在 totalIncreate < expectTotalIncrease * 90% 预警
 1. 挖矿帐号异常
    1. 是否有某个挖矿帐号， 一个小时挖矿所得为0, 可能是挖矿机器出故障了
    1. 增涨不到预期的50%
    1. 不再监控这个点： 在预期能挖到币的两倍时间间隔内， 挖到的币为0。 挖矿有波动， 在进行预警的几天里， 这个有很多误报。
 1. 数据获得失败报警
    1. result 里面 null
 1. 监控机器 heartbeat 表明监控机器没有宕机

```
{
   "id" : 1,
   "error" : null,
   "result" : {
      "blocks" : 234,
      "totalIncrease" : "4140.0000",
      "expectTotalIncrease" : "3687.5982",
      "expectBlocks" : 241,
      "minerAccounts" : [
         {
            "addr" : "1BG9ZoKtgU5bhKLpcsrncZ6xdzFCgjrZud",
            "frozen" : "30263330.0000",
            "total" : "30264474.0000",
            "expectIncrease" : "478.3413",
            "expectMinerBlocks" : "26.5745",
            "increase" : "522.0000",
            "minerBtyDuring" : "18.0000"
         },
         {
            "total" : "30286686.0000",
            "expectIncrease" : "478.6682",
            "frozen" : "30284014.0000",
            "addr" : "1KNGHukhbBnbWWnMYxu1C7YMoCj45Z3amm",
            "minerBtyDuring" : "18.0000",
            "increase" : "486.0000",
            "expectMinerBlocks" : "26.5927"
         },
         {
            "increase" : "54.0000",
            "minerBtyDuring" : "36.0000",
            "expectMinerBlocks" : "4.5225",
            "frozen" : "5150252.0000",
            "expectIncrease" : "81.4047",
            "total" : "19224972.0000",
            "addr" : "1FB8L3DykVF7Y78bRfUrRcMZwesKue7CyR"
         },
         {
            "addr" : "1Lw6QLShKVbKM6QvMaCQwTh5Uhmy4644CG",
            "total" : "30294246.0000",
            "expectIncrease" : "478.8166",
            "frozen" : "30293402.0000",
            "expectMinerBlocks" : "26.6009",
            "minerBtyDuring" : "54.0000",
            "increase" : "414.0000"
         },
         {
            "expectMinerBlocks" : "26.5927",
            "minerBtyDuring" : "36.0000",
            "increase" : "450.0000",
            "addr" : "1FiDC6XWHLe7fDMhof8wJ3dty24f6aKKjK",
            "expectIncrease" : "478.6688",
            "total" : "30288900.0000",
            "frozen" : "30284050.0000"
         },
         {
            "expectMinerBlocks" : "26.5925",
            "increase" : "594.0000",
            "minerBtyDuring" : "54.0000",
            "addr" : "1AMvuuQ7V7FPQ4hkvHQdgNWy8wVL4d4hmp",
            "frozen" : "30283780.0000",
            "expectIncrease" : "478.6645",
            "total" : "30287280.0000"
         },
         {
            "total" : "30205938.0000",
            "expectIncrease" : "25.1326",
            "frozen" : "1590072.0000",
            "addr" : "1ExRRLoJXa8LzXdNxnJvBkVNZpVw3QWMi4",
            "minerBtyDuring" : "54.0000",
            "increase" : "36.0000",
            "expectMinerBlocks" : "1.3963"
         },
         {
            "expectIncrease" : "755.1446",
            "total" : "47779640.0000",
            "frozen" : "47775904.0000",
            "addr" : "1AH9HRd4WBJ824h9PP1jYpvRZ4BSA4oN6Y",
            "minerBtyDuring" : "0.0000",
            "increase" : "576.0000",
            "expectMinerBlocks" : "41.9525"
         },
         {
            "expectMinerBlocks" : "26.5921",
            "increase" : "432.0000",
            "minerBtyDuring" : "36.0000",
            "addr" : "1PSYYfCbtSeT1vJTvSKmQvhz8y6VhtddWi",
            "frozen" : "30283312.0000",
            "total" : "30290214.0000",
            "expectIncrease" : "478.6572"
         },
         {
            "expectMinerBlocks" : "26.5834",
            "minerBtyDuring" : "36.0000",
            "increase" : "576.0000",
            "addr" : "1G7s64AgX1ySDcUdSW5vDa8jTYQMnZktCd",
            "expectIncrease" : "478.5014",
            "total" : "30278298.0000",
            "frozen" : "30273456.0000"
         }
      ],
      "seconds" : 3615
   }
}

```
