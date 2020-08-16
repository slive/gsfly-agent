/*
 * 代理服务类
 * Author:slive
 * DATE:2020/8/6
 */
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"gsfly/bootstrap"
	gch "gsfly/channel"
	"gsfly/channel/tcpx/httpx"
	gkcp "gsfly/channel/udpx/kcpx"
	logx "gsfly/logger"
)

// IService 代理服务
type IService interface {
	GetAgServer() IAgServer

	GetConf() IServiceConf

	Start() error

	Stop()

	IsClosed() bool
}

type Service struct {
	AgServer IAgServer

	ServiceConf IServiceConf

	Closed bool

	LocationHandle LocationHandle
}

func NewService(serviceConf IServiceConf, LocationHandle LocationHandle) *Service {
	if serviceConf == nil {
		errMsg := "serviceConf is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	b := &Service{ServiceConf: serviceConf}
	b.Closed = true
	b.LocationHandle = LocationHandle
	return b
}

func (b *Service) GetAgServer() IAgServer {
	return b.AgServer
}

func (service *Service) Start() error {
	id := service.GetConf().GetId()
	if !service.IsClosed() {
		return errors.New("agentservice had start, id:" + id)
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
	agServerConf := service.GetConf().GetAgServerConf()
	// TODO locationHandle处理方式？
	agServer := NewAgServer(service, agServerConf, service.LocationHandle)
	service.AgServer = agServer

	serverConf := agServerConf.GetConf()
	serverProtocol := serverConf.GetProtocol()
	var server bootstrap.IServerStrap
	switch serverProtocol {
	case gch.PROTOCOL_HTTPX:
		break
	case gch.PROTOCOL_WS:
		break
	case gch.PROTOCOL_HTTP:
		break
	case gch.PROTOCOL_KWS00:
		server = bootstrap.NewKws00Server(agServer, serverConf.(*bootstrap.KcpServerConf),
			service.onKws00AgentChannelMsgHandle, service.onKws00AgentChannelRegHandle, nil)
		// TODO 可继续注册事件
		// 默认stop事件
		chHandle := server.GetChHandle()
		chHandle.OnStopHandle = service.onAgentChannelStopHandle
		break
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

	agServer.Server = server
	err := server.Start()
	return err
}

func (ag *Service) Stop() {
	id := ag.GetConf().GetId()
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
	server := ag.GetAgServer().GetServer()
	if server != nil {
		server.Stop()
	}
}

func (b *Service) IsClosed() bool {
	return b.Closed
}

func (b *Service) GetConf() IServiceConf {
	return b.ServiceConf
}

// onAgentChannelStopHandle 当serverChannel关闭时，触发clientchannel关闭
func (b *Service) onAgentChannelStopHandle(agentChannel gch.IChannel) error {
	agentChId := agentChannel.GetId()
	defer func() {
		ret := recover()
		logx.Infof("finish to OnDstChannelStopHandle, chId:%v, ret:%v", agentChId, ret)
	}()

	logx.Info("start to onAgentChannelStopHandle, chId:", agentChId)

	return nil
}

// OnDstChannelStopHandle 当dstchannel关闭时，触发agentchannel关闭
func (b *Service) OnDstChannelStopHandle(dstChannel gch.IChannel) error {
	dstChId := dstChannel.GetId()
	defer func() {
		ret := recover()
		logx.Infof("finish to OnDstChannelStopHandle, chId:%v, ret:%v", dstChId, ret)
	}()

	// 当clientchannel关闭时，触发serverchannel关闭
	logx.Info("start to OnDstChannelStopHandle, chId:", dstChId)

	return nil
}

func (hx *Service) onKws00AgentChannelMsgHandle(agentChannel gch.IChannel, frame gkcp.Frame) error {
	r := agentChannel.GetAttach("upstream").(IUpstream)
	ctx := &UpstreamContext{
		Channel: agentChannel,
	}
	dstCh, found := r.QueryDstChannel(ctx)
	if found {
		dstPacket := dstCh.NewPacket().(*httpx.WsPacket)
		dstPacket.SetData(frame.GetPayload())
		dstPacket.MsgType = websocket.TextMessage
		dstCh.Write(dstPacket)
		agentChannel.AddAttach("opcode", frame.GetOpCode())
	}
	return nil
}

func (hx *Service) onKws00AgentChannelRegHandle(agentChannel gch.IChannel, packet gch.IPacket, attach ...interface{}) error {
	agentChId := agentChannel.GetId()
	frame, ok := attach[0].(gkcp.Frame)
	if !ok {
		return errors.New("register frame is invalid, agentChId:" + agentChId)
	}

	// TODO filter的处理
	// filterConfs := hx.GetConf().GetAgServerConf().GetFilterConfs()

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
	upstreamId := location.GetUpstreamId()
	logx.Info("upstreamId:", upstreamId)
	confMap := hx.GetConf().GetUpstreamConfs()
	ret, found := confMap[upstreamId]
	if found {
		var upstream IUpstream
		route := ret.(IUpstreamConf)
		if route.GetUpstreamType() == UPSTREAM_PROXY {
			upstream = NewProxy(hx, route.(IProxyConf))
		} else {
			// TODO
			// upstream = NewUpstream(hx, route)
		}
		upstream.SetParent(hx.GetAgServer())
		context := &UpstreamContext{
			AgServer: hx.AgServer,
			Channel:  agentChannel,
			Packet:   packet,
			Params:   make([]interface{}, 1),
		}
		context.Params[0] = params
		_, f := upstream.SelectDstChannel(context)
		if f {
			agentChannel.AddAttach("upstream", upstream)
			return nil
		}
	}
	return errors.New("select DstChannel error.")
}
