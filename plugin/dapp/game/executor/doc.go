package executor

/*
第一个区块链的游戏： 石头，剪刀，布

玩法：

1. gen(s) 算法可以是: s = hash(hash(privkey)+nonce)
2. 发起： hash(s), hash(s+石头), Lock 赌注 * 2 -> gameid (create)
3. 竞猜:  gameid， 猜想的内容，lock 赌注 (match)
4. 开奖:  公开s，申请结算 			 (close)
5. 超时： 申请赔偿 (拿到 3 * 赌注) （close）
6. 撤销： 撤销游戏 (cancel)
约束条件：限制每局最多 100 BTY，且游戏创建者所投的BTY必须是偶数

第一个步在钱包中完成，其他步骤在区块链中完成

status: Create 1 -> Match 2 -> Cancel 3 -> Close 4


//对外查询接口
//1. 我的所有赌局，按照状态进行分类 （按照地址查询）
//2. 系统所有正在进行的赌局 (按照时间进行排序)
*/
