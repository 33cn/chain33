package rpc

import (
	"fmt"
	"log"
	"types"
)

type JRpcRequest struct {
	jserver *jsonrpcServer
}
type Pdata struct {
	From  string
	TO    string
	Value string
}

//发送交易信息到topic=channel 的queue 中
func (req JRpcRequest) SendTransaction(p Pdata, result *interface{}) error {

	iclient := req.jserver.c
	message := iclient.NewMessage("channel", types.EventTx, 0, p)
	err := iclient.Send(message, true)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	log.Print("msg.id:", message.Id)
	msg, err := iclient.Wait(message.Id)
	if err != nil {
		fmt.Println("wait err:", err.Error())
		return err
	}

	*result = msg.GetData()
	return msg.Err()

}

type Qparam struct {
}

func (req JRpcRequest) QueryTransaction(p Qparam, result *interface{}) error {
	iclient := req.jserver.c
	message := iclient.NewMessage("channel", types.EventQueryTx, 0, p)
	err := iclient.Send(message, true)
	if err != nil {
		log.Fatal(err.Error())
		return err
	}

	msg, err := iclient.Wait(message.Id)
	if err != nil {
		log.Fatal("wait err:", err.Error())
		return err
	}

	*result = msg.GetData()
	return msg.Err()
}

type Gparam struct {
	Start int64
	End   int64
}

func (req JRpcRequest) GetBlocks(p Gparam, result *interface{}) error {
	iclient := req.jserver.c
	message := iclient.NewMessage("channel", types.EventGetBlocks, 0, p)
	err := iclient.Send(message, true)
	if err != nil {
		log.Fatal(err.Error())
		return err
	}

	msg, err := iclient.Wait(message.Id)
	if err != nil {
		log.Fatal("wait err:", err.Error())
		return err
	}

	*result = msg.GetData()
	return msg.Err()

}
