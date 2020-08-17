/*
 * Author:slive
 * DATE:2020/8/6
 */
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	websocket "github.com/gorilla/websocket"
	bootstrap "gsfly/bootstrap"
	gch "gsfly/channel"
	httpx "gsfly/channel/tcpx/httpx"
	gkcp "gsfly/channel/udpx/kcpx"
	common "gsfly/common"
	logx "gsfly/logger"
)

type IAgServer interface {
	common.IParent

	GetServer() bootstrap.IServerStrap

	GetConf() IAgServerConf

	Start() error

	Stop()
}

// AgServer 代理服务器
type AgServer struct {
	common.Parent

	server bootstrap.IServerStrap

	conf IAgServerConf

	// 处理location选择
	locationHandle LocationHandle
}

// NewAgServer 创建代理服务端
// parent 父节点，见IService.
// agServerConf 不可为空
func NewAgServer(parent interface{}, agServerConf IAgServerConf) *AgServer {
	if agServerConf == nil {
		err := "conf is nil"
		logx.Error(err)
		panic(err)
	}

	logx.Info("new agserver, serverconf:", agServerConf)
	b := &AgServer{conf: agServerConf}
	b.SetParent(parent)

	// 使用默认locationhandle
	b.locationHandle = defaultLocationHandle
	return b
}

func (ags *AgServer) GetServer() bootstrap.IServerStrap {
	return ags.server
}

func (ags *AgServer) GetConf() IAgServerConf {
	return ags.conf
}

func (ags *AgServer) GetLocationHandle() LocationHandle {
	return ags.locationHandle
}

func (ags *AgServer) Start() error {
	agServerConf := ags.GetConf()
	defer func() {
		ret := recover()
		if ret != nil {
			logx.Warnf("finish to agserver, id:%v, ret:%v", agServerConf.GetId(), ret)
		} else {
			logx.Info("finish to agserver, id:", agServerConf.GetId())
		}
	}()

	logx.Info("start agserver, conf:", agServerConf)
	serverConf := agServerConf.GetServerConf()
	// 根据不同协议进行不同的操作
	serverProtocol := serverConf.GetProtocol()
	var server bootstrap.IServerStrap
	switch serverProtocol {
	case gch.PROTOCOL_HTTPX:
		break
	case gch.PROTOCOL_WS:
		// httpxServer := bootstrap.NewHttpxServer(ags, serverConf.(*bootstrap.HttpxServerConf))
		// strap := httpxServer.(*bootstrap.HttpWsServerStrap)
		// strap.AddWsHandleFunc()
		break
	case gch.PROTOCOL_HTTP:
		break
	case gch.PROTOCOL_KWS00:
		server = bootstrap.NewKws00Server(ags, serverConf.(*bootstrap.KcpServerConf),
			ags.onKws00AgentChannelMsgHandle, ags.onKws00AgentChannelRegHandle, nil)
		// TODO 可继续注册事件
		// 默认stop事件
		chHandle := server.GetChHandle()
		chHandle.OnStopHandle = ags.onAgentChannelStopHandle
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
		errMsg := "unkonwn protocol, protocol:" + fmt.Sprintf("%v", serverProtocol)
		logx.Error(errMsg)
		return errors.New(errMsg)
	}

	err := server.Start()
	if err == nil {
		ags.server = server
	}
	return err
}

func (ags *AgServer) Stop() {
	logx.Info("stop agserver, conf:", ags.GetConf().GetId())
	server := ags.GetServer()
	if server != nil && !server.IsClosed() {
		server.Stop()
	}
}

const (
	Upstream_Attach_key = "upstream"
	path_key            = "path"
)

// onAgentChannelStopHandle 当serverChannel关闭时，触发clientchannel关闭
func (ags *AgServer) onAgentChannelStopHandle(agentChannel gch.IChannel) error {
	agentChId := agentChannel.GetId()
	defer func() {
		ret := recover()
		logx.Infof("finish to onAgentChannelStopHandle, chId:%v, ret:%v", agentChId, ret)
	}()
	logx.Info("start to onAgentChannelStopHandle, chId:", agentChId)
	ups := agentChannel.GetAttach(Upstream_Attach_key)
	if ups != nil {
		proxy, ok := ups.(IProxy)
		if ok {
			proxy.ClearAgentChannel(agentChannel)
		}
	}

	return nil
}

const Opcode_Key = "opcode"

func (ags *AgServer) onKws00AgentChannelMsgHandle(packet gch.IPacket) error {
	frame, ok := packet.GetAttach(gkcp.KCP_FRAME_KEY).(gkcp.Frame)
	if ok {
		agentChannel := packet.GetChannel()
		ups, found := agentChannel.GetAttach(Upstream_Attach_key).(IUpstream)
		if found {
			ctx := NewUpstreamContext(agentChannel, nil, frame)
			ups.QueryDstChannel(ctx)
			dstCh, found := ctx.GetRet(), ctx.IsOk()
			if found {
				dstPacket := dstCh.NewPacket().(*httpx.WsPacket)
				dstPacket.SetData(frame.GetPayload())
				dstPacket.MsgType = websocket.TextMessage
				dstCh.Write(dstPacket)
				agentChannel.AddAttach(Opcode_Key, frame.GetOpCode())
			}
		}
	}
	return nil
}

func (ags *AgServer) onKws00AgentChannelRegHandle(agentChannel gch.IChannel, packet gch.IPacket) error {
	agentChId := agentChannel.GetId()
	frame, ok := packet.GetAttach(gkcp.KCP_FRAME_KEY).(gkcp.Frame)
	if !ok {
		return errors.New("register frame is invalid, agentChId:" + agentChId)
	}

	// TODO filter的处理
	// filterConfs := ags.GetServerConf().GetServerConf().GetFilterConfs()

	// 步骤：
	// 先获取(可先认证)location->获取(可先认证)upstream->执行负载均衡算法->获取到clientconf
	// 获取clientchannel
	params := make(map[string]interface{})
	err := json.Unmarshal(frame.GetPayload(), &params)
	if err != nil {
		logx.Debug("params error:", err)
		return err
	}
	logx.Debug("hangup params:", params)

	// 1、约定用path来限定路径
	localPattern := params[path_key].(string)
	location := ags.locationHandle(ags, localPattern)
	if location == nil {
		s := "handle localtion error, pattern:" + localPattern
		logx.Error(s)
		return errors.New(s)
	}

	// 2、通过负载均衡获取client配置
	upstreamId := location.GetUpstreamId()
	logx.Debug("upstreamId:", upstreamId)
	upsStreams := ags.GetParent().(IService).GetUpstreams()
	ups, found := upsStreams[upstreamId]
	if found {
		context := NewUpstreamContext(agentChannel, packet, params)
		ups.SelectDstChannel(context)
		f := context.IsOk()
		if f {
			agentChannel.AddAttach(Upstream_Attach_key, ups)
			return nil
		}
	}
	return errors.New("select DstChannel error.")
}

// locationHandle 获取location，以便确认upstream的处理
// pattern 匹配路径
// params 任意参数
type LocationHandle func(server IAgServer, pattern string, params ...interface{}) ILocationConf

// defaultLocationHandle 默认LocationHandle，使用随机分配算法
func defaultLocationHandle(server IAgServer, pattern string, params ...interface{}) ILocationConf {
	aglc := defaultLocationConf
	if len(pattern) > 0 {
		locations := server.GetConf().GetLocationConfs()
		if locations != nil {
			if len(pattern) > 0 {
				lc, found := locations[pattern]
				if found {
					aglc = lc
				}
			}
		}
		if aglc == nil {
			logx.Warnf("pattern:%v, locationConf is nil", pattern)
		}
	}
	logx.Debugf("pattern:%v, locationConf:%v", pattern, aglc.GetUpstreamId())
	return aglc
}

var defaultLocationConf ILocationConf = NewLocationConf("", "", nil)
