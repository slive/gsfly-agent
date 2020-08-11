/*
 * 代理服务类
 * Author:slive
 * DATE:2020/8/6
 */
package agservice

import (
	"errors"
	"fmt"
	"gsfly/bootstrap"
	gch "gsfly/channel"
	logx "gsfly/logger"
)

type AgService interface {
	GetId() string

	GetAgServer() AgServer

	GetClientRelServerChannels() map[string]gch.Channel

	GetAgRoutes() map[string]AgRoute

	GetServerRelClients() map[string]bootstrap.ClientStrap

	Start() error
	Stop()

	IsClosed() bool
}

type BaseAgService struct {
	AgServer AgServer

	// 维护upstream产生的client和server服务器产生的channel关系
	// key为client的id，值为server产生的channel
	ClientRelServerChannels map[string]gch.Channel

	AgRoutes map[string]AgRoute

	// server服务器产生的channel和维护upstream产生的client关系
	// key为server产生的channel的id，值为client
	ServerRelClients map[string]bootstrap.ClientStrap

	Closed bool
}

func NewBaseAgService(agServer AgServer, agUpstreams ...AgRoute) *BaseAgService {
	uLen := len(agUpstreams)
	if uLen <= 0 {
		panic("upstream is nil")
	}
	b := &BaseAgService{AgServer: agServer}
	b.Closed = true
	b.ClientRelServerChannels = make(map[string]gch.Channel, 10)
	b.ServerRelClients = make(map[string]bootstrap.ClientStrap, 10)
	ups := make(map[string]AgRoute, uLen)
	for _, up := range agUpstreams {
		ups[up.GetName()] = up
	}
	b.AgRoutes = ups
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

func (b *BaseAgService) GetAgRoutes() map[string]AgRoute {
	return b.AgRoutes
}

func (b *BaseAgService) GetServerRelClients() map[string]bootstrap.ClientStrap {
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
		// hx := service.(*httpxService)
		// handleFunc := hx.serverChannelMsgHandle
		// chHandle := gch.NewChHandle(handleFunc, nil, hx.onAgentChannelStopHandle)
		// agServer := hx.AgServer.(*BaseAgServer)
		// server := bootstrap.NewKcpServer(agServer.GetServerConf().(*bootstrap.KcpServerConf), chHandle)
		// err := server.Start()
		// if err == nil {
		// 	agServer.ServerStrap = server
		// }
		return nil
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
			ch.Stop()
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

// onAgentChannelStopHandle 当serverChannel关闭时，触发clientchannel关闭
func (b *BaseAgService) onAgentChannelStopHandle(agentChannel gch.Channel) error {
	agentChId := agentChannel.GetId()
	defer func() {
		ret := recover()
		logx.Infof("finish to OnDstChannelStopHandle, chId:%v, ret:%v", agentChId, ret)
	}()

	logx.Info("start to onAgentChannelStopHandle, chId:", agentChId)
	serverRelClients := b.GetServerRelClients()
	dstClient := serverRelClients[agentChId]
	if dstClient != nil {
		delete(serverRelClients, agentChId)
		delete(b.GetClientRelServerChannels(), dstClient.GetChannel().GetId())
		dstClient.Stop()
	}
	return nil
}

// OnDstChannelStopHandle 当dstchannel关闭时，触发agentchannel关闭
func (b *BaseAgService) OnDstChannelStopHandle(dstChannel gch.Channel) error {
	dstChId := dstChannel.GetId()
	defer func() {
		ret := recover()
		logx.Infof("finish to OnDstChannelStopHandle, chId:%v, ret:%v", dstChId, ret)
	}()

	// 当clientchannel关闭时，触发serverchannel关闭
	logx.Info("start to OnDstChannelStopHandle, chId:", dstChId)
	clientRelServerChannels := b.GetClientRelServerChannels()
	serverChannel := clientRelServerChannels[dstChId]
	if serverChannel != nil {
		serverChannel.Stop()
		delete(clientRelServerChannels, dstChId)
		delete(b.GetServerRelClients(), serverChannel.GetId())
	}
	return nil
}
