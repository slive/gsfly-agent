/*
 * Author:slive
 * DATE:2020/8/6
 */
package agservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"gsfly/bootstrap"
	gch "gsfly/channel"
	gws "gsfly/channel/tcp/ws"
	gkcp "gsfly/channel/udp/kcp"
	logx "gsfly/logger"
)

type AgService interface {
	GetId() string

	GetAgServer() AgServer

	GetClientRelServerChannels() map[string]gch.Channel

	GetAgUpstreams() map[string]AgUpstream

	GetServerRelClients() map[string]bootstrap.Client

	Start() error
	Stop()

	IsClosed() bool
}

type BaseAgService struct {
	AgServer AgServer

	// 维护upstream产生的client和server服务器产生的channel关系
	// key为client的id，值为server产生的channel
	ClientRelServerChannels map[string]gch.Channel

	AgUpstreams map[string]AgUpstream

	// server服务器产生的channel和维护upstream产生的client关系
	// key为server产生的channel的id，值为client
	ServerRelClients map[string]bootstrap.Client

	Closed bool
}

func NewBaseAgService(agServer AgServer, agUpstreams ...AgUpstream) *BaseAgService {
	uLen := len(agUpstreams)
	if uLen <= 0 {
		panic("upstream is nil")
	}
	b := &BaseAgService{AgServer: agServer}
	b.Closed = true
	b.ClientRelServerChannels = make(map[string]gch.Channel, 10)
	b.ServerRelClients = make(map[string]bootstrap.Client, 10)
	ups := make(map[string]AgUpstream, uLen)
	for _, up := range agUpstreams {
		ups[up.GetName()] = up
	}
	b.AgUpstreams = ups
	return b
}

func (b *BaseAgService) GetId() string {
	return b.AgServer.GetServerConf().GetAddrStr()
}

func (b *BaseAgService) GetAgServer() AgServer {
	return b.AgServer
}

func (b *BaseAgService) GetClientRelServerChannels() map[string]gch.Channel {
	return b.ClientRelServerChannels
}

func (b *BaseAgService) GetAgUpstreams() map[string]AgUpstream {
	return b.AgUpstreams
}

func (b *BaseAgService) GetServerRelClients() map[string]bootstrap.Client {
	return b.ServerRelClients
}

func (b *BaseAgService) Start() error {

	return nil
}

func Start(service AgService) error {
	id := service.GetId()
	if !service.IsClosed() {
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
	serverConf := service.GetAgServer().GetServerConf()
	serverProtocol := serverConf.GetProtocol()
	switch serverProtocol {
	case gch.PROTOCOL_HTTPX:
		break
	case gch.PROTOCOL_WS:
		break
	case gch.PROTOCOL_HTTP:
		break
	case gch.PROTOCOL_KWS00:
		hx := service.(*httpxService)
		handleFunc := hx.HandleServerChannelMsg
		chHandle := gch.NewChHandle(handleFunc, nil, hx.HandleServerChannelClose)
		agServer := hx.AgServer.(*BaseAgServer)
		server := bootstrap.NewKcpServer(agServer.GetServerConf().(*bootstrap.KcpServerConf), chHandle)
		err := server.Start()
		if err == nil {
			agServer.Server = server
		}
		return err
	case gch.PROTOCOL_KCP:
		break
	case gch.PROTOCOL_TCP:
		break
	case gch.PROTOCOL_UDP:
		break
	case gch.PROTOCOL_KWS01:
		break
	default:
		return errors.New("unkonwn protocol, protocol:" + fmt.Sprintf("%v", serverProtocol))
	}
	return nil
}

func (ag *BaseAgService) Stop() {
	id := ag.GetId()
	if ag.IsClosed() {
		logx.Info("had close agent, id:", id)
		return
	}

	defer func() {
		ret := recover()
		if ret != nil {
			logx.Errorf("finish to close agent, id:%v, msg:%v", id, ret)
		} else {
			logx.Info("finish to close agent, id:", id)
		}
	}()
	logx.Info("start to close agent, id:", id)
	dstChannels := ag.GetClientRelServerChannels()
	if dstChannels != nil {
		for key, ch := range dstChannels {
			ch.StopChannel(ch)
			delete(dstChannels, key)
		}
	}

	srcChannels := ag.GetServerRelClients()
	if srcChannels != nil {
		for key, _ := range srcChannels {
			delete(srcChannels, key)
		}
	}
	server := ag.GetAgServer().GetServer()
	if server != nil {
		server.Stop()
	}
}

func (b *BaseAgService) IsClosed() bool {
	return b.Closed
}

// HandleServerChannelClose 当serverChannel关闭时，触发clientchannel关闭
func (b *BaseAgService) HandleServerChannelClose(serverChannel gch.Channel) error {
	serverChId := serverChannel.GetChId()
	defer func() {
		ret := recover()
		logx.Infof("finish to HandleClientChannelClose, chId:%v, ret:%v", serverChId, ret)
	}()
	logx.Info("start to HandleServerChannelClose, chId:", serverChId)
	serverRelClients := b.GetServerRelClients()
	dstClient := serverRelClients[serverChId]
	if dstClient != nil {
		delete(serverRelClients, serverChId)
		delete(b.GetClientRelServerChannels(), dstClient.GetChannel().GetChId())
		dstClient.Stop()
	}
	return nil
}

// HandleClientChannelClose 当clientchannel关闭时，触发serverchannel关闭
func (b *BaseAgService) HandleClientChannelClose(clientChannel gch.Channel) error {
	clientChId := clientChannel.GetChId()
	defer func() {
		ret := recover()
		logx.Infof("finish to HandleClientChannelClose, chId:%v, ret:%v", clientChId, ret)
	}()
	// 当clientchannel关闭时，触发serverchannel关闭
	logx.Info("start to HandleClientChannelClose, chId:", clientChId)
	clientRelServerChannels := b.GetClientRelServerChannels()
	serverChannel := clientRelServerChannels[clientChId]
	if serverChannel != nil {
		serverChannel.StopChannel(serverChannel)
		delete(clientRelServerChannels, clientChId)
		delete(b.GetServerRelClients(), serverChannel.GetChId())
	}
	return nil
}

type httpxService struct {
	BaseAgService
}

func NewHttpxAgService(agServer AgServer, agUpstreams ...AgUpstream) *httpxService {
	b := &httpxService{}
	b.BaseAgService = *NewBaseAgService(agServer, agUpstreams...)
	return b
}

func (hx *httpxService) Start() error {
	return Start(hx)
}

func (hx *httpxService) HandleServerChannelMsg(packet gch.Packet) error {
	defer func() {
		err := recover()
		if err != nil {
			logx.Error("handle error:", err)
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
				// 步骤：先获取location->获取到upstream->执行负载均衡算法->获取到clientconf
				params := make(map[string] interface{})
				err := json.Unmarshal(frame.GetPayload(), &params)
				if err != nil {
					logx.Debug("params error:", err)
					return err
				}
				logx.Info("hangup params:",params)

				localPattern := params["url"]
				handleLocation := hx.AgServer.GetHandleLocation()
				location := handleLocation(hx.GetAgServer(), localPattern)

				upstreamName := location.GetUpstreamName()
				agUpstream := hx.GetAgUpstreams()[upstreamName]
				handleBalance := agUpstream.GetHandleLoadBalance()
				clientConf := handleBalance(agUpstream)

				handle := gch.NewChHandle(hx.HandleClientChannelMsg, nil, hx.HandleClientChannelClose)
				wsClientConf := clientConf.(*bootstrap.WsClientConf)
				// err := json.Unmarshal(frame.GetPayload(), &wsClientConf.Params)
				wsClientConf.Params = params

				logx.Info("params:", wsClientConf.Params)
				wsClient := bootstrap.NewWsClient(wsClientConf, handle)
				err = wsClient.Start()
				if err != nil {
					logx.Info("dialws error, srcChId:" + srcChId)
					return err
				}
				// 拨号成功，记录
				dstCh := wsClient.GetChannel()
				dstChId := dstCh.GetChId()
				hx.GetServerRelClients()[srcChId] = wsClient
				hx.GetClientRelServerChannels()[dstChId] = srcCh
				logx.Info("dstChId:" + dstChId + ", srcChId:" + srcChId)
				return err
			} else {
				dstCh := hx.GetServerRelClients()[srcChId].GetChannel()
				if dstCh != nil {
					dstPacket := dstCh.NewPacket().(*gws.WsPacket)
					dstPacket.SetData(frame.GetPayload())
					dstPacket.MsgType = websocket.TextMessage
					dstCh.Write(dstPacket)
				}
			}
		} else {
			logx.Info("frame is nil")
		}
	} else {
		// TODO ?
	}
	return nil
}

func (hx *httpxService) HandleClientChannelMsg(packet gch.Packet) error {
	dstChId := packet.GetChannel().GetChId()
	srcCh := hx.GetClientRelServerChannels()[dstChId]
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
