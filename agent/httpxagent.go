/*
 * Author:slive
 * DATE:2020/8/5
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
)

type KwsToHttpxAgent struct {
	BaseAgent
	// ServerConf *strap.KcpServerConf
	// ClientConfMap map[string]*strap.WsClientConf
	// ClientConfs   []*strap.WsClientConf
}

func NewKwsToHttpx(serverConf *strap.KcpServerConf, clientConfs ...*strap.WsClientConf) Agent {
	clen := len(clientConfs)
	if clen < 0 {
		logx.Error("client conf is nil")
		return nil
	}
	kh := &KwsToHttpxAgent{}
	wsClientConfs := make([]strap.ClientConf, clen)
	for index, cc := range clientConfs {
		wsClientConfs[index] = cc
	}
	kh.BaseAgent = *NewBaseAgent(serverConf, DefalutSelectHandle, wsClientConfs...)
	return kh
}

func (ka *KwsToHttpxAgent) Start() error {
	id := ka.GetId()
	if !ka.IsClosed() {
		return errors.New("agent had start, id:" + id)
	}

	defer func() {
		ret := recover()
		if ret != nil {
			logx.Errorf("finish to start agent, id:%v, msg:%v", id, ret)
		} else {
			logx.Info("finish to start agent, id:", id)
		}
	}()
	logx.Info("start to agent, id:", id)
	handleFunc := ka.srcHandleFunc
	chHandle := gch.NewChHandle(handleFunc, nil, ka.SrcCloseFunc)
	server := strap.NewKcpServer(ka.GetServerConf().(*strap.KcpServerConf), chHandle)
	err := server.Start()
	if err == nil {
		ka.Server = server
	}
	return err
}

func (ka *KwsToHttpxAgent) Stop() {
	server := ka.Server
	if server != nil {
		server.Stop()
	}
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
				handle := gch.NewChHandle(ka.dstHandleFunc, nil, ka.DstCloseFunc)
				clientConf := ka.SelectHandle(ka)
				wsClientConf := clientConf.(*strap.WsClientConf)
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
				ka.DstChannels[srcChId] = dstCh
				ka.SrcChannels[dstChId] = srcCh
				glog.Info("dstChId:" + dstChId + ", srcChId:" + srcChId)
				return err
			} else {
				dstCh := ka.DstChannels[srcChId]
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
	srcCh := ka.SrcChannels[dstChId]
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
