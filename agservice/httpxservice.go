/*
 * Author:slive
 * DATE:2020/8/6
 */
package agservice

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"gsfly/bootstrap"
	gch "gsfly/channel"
	httpx "gsfly/channel/tcpx/httpx"
	gkcp "gsfly/channel/udpx/kcpx"
	logx "gsfly/logger"
)

type httpxService struct {
	BaseAgService
}

func NewHttpxAgService(agServer AgServer, agUpstreams ...AgRoute) *httpxService {
	b := &httpxService{}
	b.BaseAgService = *NewBaseAgService(agServer, agUpstreams...)
	return b
}

func (hx *httpxService) Start() error {
	id := hx.GetId()
	if !hx.IsClosed() {
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
	logx.Info("start to agent service, id:", id)

	agServer := hx.AgServer.(*BaseAgServer)
	protocol := agServer.ServerConf.GetProtocol()
	var server bootstrap.ServerStrap
	switch protocol {
	case gch.PROTOCOL_WS:
		// handleFunc := hx.serverChannelMsgHandle
		// chHandle := gch.NewDefaultChHandle(handleFunc)
		// chHandle.OnStopHandle = hx.onAgentChannelStopHandle
		// server = bootstrap(agServer.GetServerConf().(*bootstrap.KcpServerConf), chHandle)
		break
	case gch.PROTOCOL_KWS00:
		server = bootstrap.NewKws00Server(hx, agServer.GetServerConf().(*bootstrap.KcpServerConf),
			hx.onKws00AgentChannelMsgHandle, hx.onKws00ServerAgentRegHandle, nil)
		// TODO 可继续注册事件
		// 默认stop事件
		chHandle := server.GetChannelHandle()
		chHandle.OnStopHandle = hx.onAgentChannelStopHandle
		break
	}

	err := server.Start()
	if err == nil {
		agServer.Server = server
	}
	return err
}

func (hx *httpxService) onKws00ServerAgentRegHandle(agentChannel gch.Channel, packet gch.Packet, attach ...interface{}) error {
	agentChId := agentChannel.GetId()
	frame, ok := attach[0].(gkcp.Frame)
	if !ok {
		return errors.New("register frame is invalid, agentChId:" + agentChId)
	}

	// 步骤：
	// 先获取(可先认证)location->获取(可先认证)upstream->执行负载均衡算法->获取到clientconf
	// 获取clientchannel
	params := make(map[string]interface{})
	err := json.Unmarshal(frame.GetPayload(), &params)
	if err != nil {
		logx.Debug("params error:", err)
		return err
	}
	logx.Info("hangup params:", params)

	// 1、约定用path来限定路径
	localPattern := params["path"].(string)
	handleLocation := hx.AgServer.GetLocationHandle()
	location := handleLocation(hx.GetAgServer(), localPattern)
	if location == nil {
		s := "handle localtion error, pattern:" + localPattern
		logx.Error(s)
		return errors.New(s)
	}

	// 2、通过负载均衡获取client配置
	routeName := location.GetRouteName()
	agUpstream := hx.GetAgRoutes()[routeName]
	if agUpstream == nil {
		s := "get agUpstream error, routeName:" + routeName
		logx.Error(s)
		return errors.New(s)
	}
	lbcontext := NewLoadBalanceContext(nil, agUpstream, agentChannel)
	handleBalance := agUpstream.GetHandleLoadBalance()
	handleBalance(lbcontext)
	clientConf := lbcontext.Result
	if clientConf == nil {
		s := "handle loadbalance error, routeName:" + routeName
		logx.Error(s)
		return errors.New(s)
	}

	// 3、代理到目标
	clientPro := clientConf.GetProtocol()
	switch clientPro {
	case gch.PROTOCOL_WS, gch.PROTOCOL_HTTPX:
		handle := gch.NewDefaultChHandle(hx.onDstChannelMsgHandle)
		handle.OnStopHandle = hx.OnDstChannelStopHandle
		wsClientConf := clientConf.(*bootstrap.WsClientConf)
		wsClientConf.Params = params

		logx.Info("params:", wsClientConf.Params)
		wsClient := bootstrap.NewWsClient(hx, wsClientConf, handle)
		err = wsClient.Start()
		if err != nil {
			logx.Info("dialws error, agentChId:" + agentChId)
			return err
		}
		// 拨号成功，记录
		dstCh := wsClient.GetChannel()
		dstChId := dstCh.GetId()
		hx.GetServerRelClients()[agentChId] = wsClient
		hx.GetClientRelServerChannels()[dstChId] = agentChannel
		logx.Info("dstChId:" + dstChId + ", agentChId:" + agentChId)
		break
	case gch.PROTOCOL_HTTP:
		break
	case gch.PROTOCOL_KWS00:
		break
	case gch.PROTOCOL_KWS01:
		break
	case gch.PROTOCOL_TCP:
		break
	case gch.PROTOCOL_UDP:
		break
	case gch.PROTOCOL_KCP:
		break
	default:
		// gch.PROTOCOL_WS
	}

	return err
}

// onKws00AgentChannelMsgHandle 代理服务端的channel收到消息后的处理
func (hx *httpxService) onKws00AgentChannelMsgHandle(agentChannel gch.Channel, frame gkcp.Frame) error {
	agentChId := agentChannel.GetId()
	dstCh := hx.GetServerRelClients()[agentChId].GetChannel()
	logx.Info("agentChId:", agentChId)
	if dstCh != nil {
		dstPacket := dstCh.NewPacket().(*httpx.WsPacket)
		dstPacket.SetData(frame.GetPayload())
		dstPacket.MsgType = websocket.TextMessage
		dstCh.Write(dstPacket)
		agentChannel.AddAttach("opcode", frame.GetOpCode())
	}
	return nil
}

// onDstChannelMsgHandle upstream的客户端channel收到消息后的处理，直接会写到server对应的客户端channel
func (hx *httpxService) onDstChannelMsgHandle(packet gch.Packet) error {
	dstChannel := packet.GetChannel()
	dstChId := dstChannel.GetId()
	agentChannel := hx.GetClientRelServerChannels()[dstChId]
	logx.Info("dstChId:", dstChId)
	if agentChannel != nil {
		// 回写，区分第一次，最后一次等？
		opcode := gkcp.OPCODE_TEXT_SIGNALLING
		oc := agentChannel.GetAttach("opcode")
		if oc != nil {
			cpc, ok := oc.(uint16)
			if ok {
				opcode = cpc
			}
		}
		frame := gkcp.NewOutputFrame(opcode, packet.GetData())
		srcPacket := agentChannel.NewPacket()
		srcPacket.SetData(frame.GetKcpData())
		agentChannel.Write(srcPacket)
		return nil
	} else {
		s := "src channel is nil, dst channel id:" + dstChId
		logx.Error(s)
		return errors.New(s)
	}
}
