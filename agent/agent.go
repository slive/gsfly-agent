/*
 * Author:slive
 * DATE:2020/7/28
 */
package agent

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	logx "github.com/sirupsen/logrus"
	strap "gsfly/bootstrap"
	gch "gsfly/channel"
	gws "gsfly/channel/tcp/ws"
	gkcp "gsfly/channel/udp/kcp"
	glog "gsfly/logger"
	"math/rand"
	"unsafe"
)

// agent
//  server
//      network
//      listen
//      config
//
//
//  client
//      network
//      config
//
//      upstream

type Agent interface {
}

type BaseAgent struct {
	serverConf    *strap.KcpServerConf
	clientConfMap map[string]*strap.ClientConf
	clientConfs   []*strap.ClientConf
	dstChannels   map[string]gch.Channel
	srcChannels   map[string]gch.Channel
	// 路由规则
	SelectHandle SelectHandle
	QueryHandle  QueryHandle
}

type KwsToHttpxAgent struct {
	BaseAgent
	serverConf    *strap.KcpServerConf
	clientConfMap map[string]*strap.WsClientConf
	clientConfs   []*strap.WsClientConf
}

func KwsToHttpx(serverConf *strap.KcpServerConf, clientConfs ...*strap.WsClientConf) {
	kh := &KwsToHttpxAgent{
		serverConf:    serverConf,
		clientConfMap: make(map[string]*strap.WsClientConf, len(clientConfs)),
	}
	kh.dstChannels = make(map[string]gch.Channel, 100)
	kh.srcChannels = make(map[string]gch.Channel, 100)
	kh.clientConfs = clientConfs
	for _, cc := range clientConfs {
		kh.clientConfMap[cc.GetAddrStr()] = cc
	}

	kh.SelectHandle = DefalutSelectHandle

	handleFunc := kh.srcHandleFunc
	// TODO 加密
	chHandle := gch.NewChHandle(handleFunc, nil, kh.srcCloseFunc)
	server := strap.NewKcpServer(serverConf, chHandle)
	server.Start()
}

func (ka *KwsToHttpxAgent) srcCloseFunc(channel gch.Channel) error {
	// 错误处理
	id := channel.GetChId()
	glog.Info("do srcclose, chId:", id)
	dstChannel := ka.dstChannels[id]
	dstChannel.StopChannel(dstChannel)
	return nil
}

func (ka *KwsToHttpxAgent) srcHandleFunc(packet gch.Packet) error {
	defer func() {
		err := recover()
		if err != nil {
			glog.Error("handle error:", err)
		}
	}()
	srcCh := packet.GetChannel()
	// 强制转换处理
	kwsPacket, ok := packet.(*gkcp.KwsPacket)
	if ok {
		frame := kwsPacket.Frame
		if frame != nil {
			srcChId := srcCh.GetChId()
			// 第一次建立会话时进行处理
			if frame.GetOpCode() == gkcp.OPCODE_TEXT_SESSION {
				handle := gch.NewChHandle(ka.dstHandleFunc, nil, ka.dstCloseFunc)
				clientConf := ka.SelectHandle(ka)
				wsClientConf := (*strap.WsClientConf)(unsafe.Pointer(clientConf))
				err := json.Unmarshal(frame.GetPayload(), &wsClientConf.Params)
				if err != nil {
					glog.Debug("params error:", err)
					return err
				}
				glog.Info("params:", wsClientConf.Params)
				wsClient := strap.NewWsClient(wsClientConf, handle)
				err = wsClient.Start()
				if err != nil {
					glog.Info("dialws error, srcChId:" + srcChId)
					return err
				}
				// 拨号成功，记录
				dstCh := wsClient.GetChannel()
				dstChId := dstCh.GetChId()
				ka.dstChannels[srcChId] = dstCh
				ka.srcChannels[dstChId] = srcCh
				glog.Info("dstChId:" + dstChId + ", srcChId:" + srcChId)
				return err
			} else {
				dstCh := ka.dstChannels[srcChId]
				if dstCh != nil {
					dstPacket := dstCh.NewPacket().(*gws.WsPacket)
					dstPacket.SetData(frame.GetPayload())
					dstPacket.MsgType = websocket.TextMessage
					dstCh.Write(dstPacket)
				}
			}
		} else {
			glog.Info("frame is nil")
		}
	} else {
		// TODO ?
	}
	return nil
}

func (ka *KwsToHttpxAgent) dstHandleFunc(packet gch.Packet) error {
	dstChId := packet.GetChannel().GetChId()
	srcCh := ka.srcChannels[dstChId]
	if srcCh != nil {
		// 回写
		frame := gkcp.NewOutputFrame(gkcp.OPCODE_TEXT_SIGNALLING, packet.GetData())
		srcPacket := srcCh.NewPacket()
		srcPacket.SetData(frame.GetKcpData())
		srcCh.Write(srcPacket)
		return nil
	} else {
		return errors.New("src channel is nil, dst channel id:" + dstChId)
	}
}

func (ka *KwsToHttpxAgent) dstCloseFunc(channel gch.Channel) error {
	// 错误处理
	id := channel.GetChId()
	glog.Info("do dstclose, chId:", id)
	srcChannel := ka.srcChannels[id]
	srcChannel.StopChannel(srcChannel)
	return nil
}

func DefalutSelectHandle(agent Agent) *strap.WsClientConf {
	k := agent.(*KwsToHttpxAgent)
	ccLen := len(k.clientConfs)
	rand := rand.Int() % ccLen
	conf := k.clientConfs[rand]
	logx.Infof("rand:%v, conf:%v", rand, conf)
	return conf
}

type SelectHandle func(agent Agent) *strap.WsClientConf

type QueryHandle func(agent Agent, client *strap.WsClientConf) strap.Client
