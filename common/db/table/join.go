package table

/*
设计表的联查:

我们不可能做到数据库这样强大的功能，但是联查功能几乎是不可能绕过的功能。

table1:

[gameId, status]

table2:
[txhash, gameId, addr]

他们都独立的构造, 与更新

如果设置了两个表的: join, 比如: addr & status 要作为一个查询key, 那么我们需要维护一个:

join_table2_table1:
//table2 primary key
//table1 primary key
//addr_status 的一个关联index
[txhash, gameId, addr_status]

能够join的前提：
table2 包含 table1 的primary key

数据更新:
table1 数据更新 自动触发: join_table2_table1 更新 addr & status
table2 数据更新 也会自动触发: join_table2_table1 更新 addr & status

例子:

table1 更新了 gameId 对应 status -> 触发 join_table2_table1 所有对应 gameId 更新 addr & status
table2 更新了 txhash 对应的 addr -> 触发 join_table2_table1 所有对应的 txhash 对应的 addr & status

注意 join_table2_table1 是自动维护的

table2 中自动可以查询 addr & status 这个index
*/
