## 挖矿监控

数据获得
```
wget 127.0.0.1:8866 --no-proxy --post-data='{"id" : 1 , "method" : "ShowMinerAccount.Get", "params" :  [{"timeAt" : ""}] } '
```

结果：
 1. seconds 里是实际挖矿时间。 理论上返回是一小时， 但实际上可能是有误差的
 2. totalIncreate 这段时间挖矿的所有币
 3. minerAccounts 一共配置里的10个挖矿帐号信息
 4. increase 是对应挖矿帐号的这段时间挖矿所得
 5. total 是对应挖矿帐号里的所有的币
 6. addr 是对应挖矿帐号的地址

需要监控
 1. 是否有某个挖矿帐号， 一个小时挖矿所得为0, 可能是挖矿机器出故障了
 1. 挖矿总量异常： 我看现在的挖矿量也是有波动的从一个小时 3700多到4600多不等。 以后随着其他客户也参与挖矿， 可能挖矿量也会有所下降。 监控这个数据小于 3330 为异常。 以后看挖矿情况调整。
```
{
   "result" : {
      "seconds" : 3630,
      "totalIncrease" : "4158.0000",
      "minerAccounts" : [
         {
            "increase" : "720.0000",
            "total" : "47772386.0000",
            "addr" : "1AH9HRd4WBJ824h9PP1jYpvRZ4BSA4oN6Y"
         },
         {
            "increase" : "540.0000",
            "total" : "30289872.0000",
            "addr" : "1Lw6QLShKVbKM6QvMaCQwTh5Uhmy4644CG"
         },
         {
            "increase" : "522.0000",
            "addr" : "1PSYYfCbtSeT1vJTvSKmQvhz8y6VhtddWi",
            "total" : "30285858.0000"
         },
         {
            "total" : "30205650.0000",
            "addr" : "1ExRRLoJXa8LzXdNxnJvBkVNZpVw3QWMi4",
            "increase" : "54.0000"
         },
         {
            "increase" : "504.0000",
            "addr" : "1KNGHukhbBnbWWnMYxu1C7YMoCj45Z3amm",
            "total" : "30282492.0000"
         },
         {
            "increase" : "450.0000",
            "total" : "30282744.0000",
            "addr" : "1AMvuuQ7V7FPQ4hkvHQdgNWy8wVL4d4hmp"
         },
         {
            "total" : "19224252.0000",
            "addr" : "1FB8L3DykVF7Y78bRfUrRcMZwesKue7CyR",
            "increase" : "36.0000"
         },
         {
            "total" : "30260280.0000",
            "addr" : "1BG9ZoKtgU5bhKLpcsrncZ6xdzFCgjrZud",
            "increase" : "396.0000"
         },
         {
            "addr" : "1G7s64AgX1ySDcUdSW5vDa8jTYQMnZktCd",
            "total" : "30274482.0000",
            "increase" : "486.0000"
         },
         {
            "total" : "30284526.0000",
            "addr" : "1FiDC6XWHLe7fDMhof8wJ3dty24f6aKKjK",
            "increase" : "450.0000"
         }
      ]
   },
   "id" : 1,
   "error" : null
}
```
