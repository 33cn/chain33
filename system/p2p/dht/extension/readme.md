# pubsub接口说明与介绍
````
pubsub是libp2p的订阅与发布功能，网络中的节点可以订阅其感兴趣的topic也可以称之为订阅主题
一个节点可以向主题(topic)发送消息，每个发送的消息都可以传递给所有订阅该主题的节点。
````
### pubsub的特点

````
在pub/sub系统中，所有的节点都参与网络消息的传递。pub/sub系统有几种不同的设计，
可以提供不同的权衡，包含如下属性：
````
* 可靠性
  > 所有的消息都要传递给订阅了该主题的节点。
* 传播速度
  > 快速的传递消息。
* 传播效率
  > 网络不会被过多的消息副本淹没。
* 弹性
  > 节点可以加入并离开网络，而不会引起网络消息的中断。
* 规模
  > 可以满足大量的节点订阅相同的消息主题
* 简便性
  > 该系统设计的要易于理解和实施
        

### pubsub在chain33中用途
````
pubsub基础功能集成在chain33/system/p2p/dht/net/pubsub.go中,该功能主要包含订阅topic,向
topic 发送消息，删除主题，获取已经订阅的topic.

目前pubsub功能刷线应用在chain33平行链模块，主要应用于在平行链内部共识期间的消息广播。
chain33平行链使用相关功能，需要先通过queue消息处理模块 发送相关消息事件注册自己的模块名与要订阅的topic，
当有相关topic消息过来之后，pubsub就会通过queue 把接收到消息转发给注册的相应模块。
后期计划用pubsub去掉chain33的tx广播功能。
````
#### 接口说明

1 创建pubsub对象


```gotemplate
func NewPubSub(ctx context.Context, host host.Host) (*PubSub, error) 
```


2  订阅topic

```gotemplate

func (p *PubSub) JoinTopicAndSubTopic(topic string, callback subCallBack, opts ...pubsub.TopicOpt) error 
``` 
- callback是一个回调函数指针，用于注册接收订阅消息后的后续处理。

3 发布消息

```gotemplate
func (p *PubSub) Publish(topic string, msg []byte) error
```

- topic 指定相应的topic
- msg 要发布的消息


4 删除topic

```gotemplate
func (p *PubSub) RemoveTopic(topic string) 
```

5 获取订阅相关topic的peer列表

```gotemplate
func (p *PubSub) FetchTopicPeers(topic string) []peer.ID 
```

6. 查询订阅的topic 列表

```gotemplate
func (p *PubSub) GetTopics() []string 
```



# Relay接口说明与介绍

```
在去中心化场景下，我们总是希望节点能够直接与其他节点进行通信。然而实际上，许多节点是无法访问的，因为它们可能存在 NAT 穿透问题，或使用的平台不允许外部访问。
为了解决这个问题， libp2p 提供了一种名为 relay 的协议，允许节点充当另外两个节点的 proxy 代理。所有的通信都经过加密，并且经过远程身份认证，所以代理服务不会遭遇中间人攻击（ man-in-the-middle ）。

```

### Relay的特点

```
Relay 功能需要一些有中继功能的节点作为proxy。

```

### pubsub在chain33中用途
````
中继功能目前还没有实际应用在chain33中，作为探索开发，为以后chain33有实际需求后，提供储备。
````


#### 接口说明 

1. 创建Relay 对象

````gotemplate
func NewRelayDiscovery(host host.Host, adv *discovery.RoutingDiscovery, opts ...circuit.RelayOpt) *Relay 
````
- opts 在创建Relay对象的时候，需要设置RelayOption 如果计划自身节点作为中继节点则需要设置为RelayHop，用户接受其他节点
发过来的中继请求，中继服务端必须配置。RelayDiscovery 作为客户端请求放必须配置。

```gotemplate

                peer:a,b,c 
                a config RelayDiscovery
                b config RelayHop,RelayActive
                c config nothing
                a->b a connect b
                a.DialPeer(b,c) will success
                
                if b just config RelayHop,a->b a connect b
                a.DialPeer(b,c) will failed
    
                if  b just config RelayHop,a->b,b->c 
                a.DialPeer(b,c) will success
```
             

2  发现中继节点

```gotemplate
func (r *Relay) FindOpPeers() ([]peer.AddrInfo, error) 
```

3 通过hop中继节点连接dest 目标节点

````gotemplate
func (r *Relay) DialDestPeer(host host.Host, hop, dst peer.AddrInfo) (*circuit.Conn, error)
````

4 检查节点是否是中继节点

````gotemplate
func (r *Relay) CheckHOp(host host.Host, isop peer.ID) (bool, error) 
````
