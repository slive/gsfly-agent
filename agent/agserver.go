/*
 * Author:slive
 * DATE:2020/8/6
 */
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	bootstrap "github.com/Slive/gsfly/bootstrap"
	gch "github.com/Slive/gsfly/channel"
	httpx "github.com/Slive/gsfly/channel/tcpx/httpx"
	gkcp "github.com/Slive/gsfly/channel/udpx/kcpx"
	common "github.com/Slive/gsfly/common"
	logx "github.com/Slive/gsfly/logger"
	websocket "github.com/gorilla/websocket"
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
// AgServerConf 不可为空
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
	var serverStrap bootstrap.IServerStrap
	switch serverProtocol {
	case gch.PROTOCOL_HTTPX:
	case gch.PROTOCOL_WS:
		chHandle := gch.NewDefChHandle(ags.onAgentChannelMsgHandle)
		chHandle.SetOnRegisteredHandle(ags.onAgentChannelRegHandle)
		wsServerConf := (bootstrap.IServerConf(serverConf)).(*bootstrap.WsServerConf)
		wsServerStrap := bootstrap.NewWsServerStrap(ags, wsServerConf, chHandle, nil)
		serverStrap = wsServerStrap
	case gch.PROTOCOL_HTTP:
	case gch.PROTOCOL_KWS00:
		kwsServerStrap := bootstrap.NewKws00ServerStrap(ags, serverConf,
			ags.onAgentChannelMsgHandle, ags.onAgentChannelRegHandle, nil)
		serverStrap = kwsServerStrap
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

	if serverStrap != nil {
		// TODO 可继续注册事件
		// 默认stop事件
		chHandle := serverStrap.GetChHandle()
		chHandle.OnStopHandle = ags.onAgentChannelStopHandle
		err := serverStrap.Start()
		if err == nil {
			ags.server = serverStrap
		}
		return err
	}
	errMs := "serverStrap is nil"
	logx.Error(errMs)
	return errors.New(errMs)
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

func (ags *AgServer) onAgentChannelMsgHandle(packet gch.IPacket) error {
	agentChannel := packet.GetChannel()
	ups, found := agentChannel.GetAttach(Upstream_Attach_key).(IUpstream)
	if found {
		ctx := NewUpstreamContext(agentChannel, packet)
		ups.QueryDstChannel(ctx)
		dstCh, found := ctx.GetRet(), ctx.IsOk()
		if found {
			ProxyWrite(packet, dstCh)
			return nil
		}
	}
	logx.Warn("unknown agent ProxyWrite.")
	return nil
}

func ProxyWrite(fromPacket gch.IPacket, toChannel gch.IChannel) {
	fromChannel := fromPacket.GetChannel()
	dstPacket := toChannel.NewPacket()
	fromProtocol := fromChannel.GetConf().GetProtocol()
	toProtocol := toChannel.GetConf().GetProtocol()
	activating := fromChannel.GetAttach(Activating_Key)
	logx.Debugf("from chId:%v, activating:%v", fromChannel.GetId(), activating)
	// 根据不同的协议类型，转发到不同的目的dstChannel
	switch fromProtocol {
	case gch.PROTOCOL_WS:
		switch toProtocol {
		case gch.PROTOCOL_KWS00:
			opCode := gkcp.OPCODE_TEXT_SIGNALLING
			if activating != nil {
				at, f := activating.(bool)
				if f && at {
					opCode = gkcp.OPCODE_TEXT_SESSION
				}
			}

			frame := gkcp.NewOutputFrame(opCode, fromPacket.GetData())
			dstPacket.(*gkcp.KWS00Packet).Frame = frame
			dstPacket.SetData(frame.GetKcpData())
			// 用于代理回复后对应
			fromChannel.AddAttach(Opcode_Key, frame.GetOpCode())
		case gch.PROTOCOL_WS:
			dstPacket.SetData(fromPacket.GetData())
			dstPacket.(*httpx.WsPacket).MsgType = fromPacket.(*httpx.WsPacket).MsgType
		default:
			dstPacket.SetData(fromPacket.GetData())
		}
	case gch.PROTOCOL_KWS00:
		frame := fromPacket.(*gkcp.KWS00Packet).Frame
		switch toProtocol {
		case gch.PROTOCOL_KWS00:
			dstPacket.SetData(frame.GetKcpData())
		case gch.PROTOCOL_WS:
			dstPacket.SetData(frame.GetPayload())
			dstPacket.(*httpx.WsPacket).MsgType = websocket.TextMessage
		default:
			dstPacket.SetData(fromPacket.GetData())
		}
		// 用于代理回复后对应
		fromChannel.AddAttach(Opcode_Key, frame.GetOpCode())
	default:
		dstPacket.SetData(fromPacket.GetData())
	}
	toChannel.Write(dstPacket)
	fromChannel.AddAttach(Activating_Key, false)
}

func (ags *AgServer) GetLocationPattern(agentChannel gch.IChannel, packet gch.IPacket) (localPattern string, params []interface{}) {
	protocol := agentChannel.GetConf().GetProtocol()
	localPattern = ""
	params = make([]interface{}, 1)
	switch protocol {
	case gch.PROTOCOL_WS:
		wsChannel := agentChannel.(*httpx.WsChannel)
		params[0] = wsChannel.GetParams()
		// 1、约定用path来限定路径
		wsServerConf := wsChannel.GetConf().(bootstrap.IWsServerConf)
		localPattern = wsServerConf.GetPath()
	case gch.PROTOCOL_KWS00:
		agentChId := agentChannel.GetId()
		frame, ok := packet.GetAttach(gkcp.KCP_FRAME_KEY).(gkcp.Frame)
		if !ok {
			logx.Warn("register frame is invalid, agentChId:", agentChId)
		} else {
			kws00Params := make(map[string]interface{})
			err := json.Unmarshal(frame.GetPayload(), &kws00Params)
			if err != nil {
				logx.Warn("params error:", err)
			} else {
				// 1、约定用path来限定路径
				path := kws00Params[path_key]
				if path == nil {
					localPattern = ""
				} else {
					localPattern = fmt.Sprintf("%v", path)
				}
				params[0] = kws00Params
			}
		}
	default:
		// TODO 待完善
	}
	logx.Infof("channel params:%v, localPattern:%v", params, localPattern)
	return localPattern, params
}

const Activating_Key = "activating"

func (ags *AgServer) onAgentChannelRegHandle(agentChannel gch.IChannel, packet gch.IPacket) error {
	// TODO filter的处理
	// FilterConfs := ags.GetServerConf().GetServerConf().GetFilterConfs()

	// 步骤：
	// 先获取(可先认证)location->获取(可先认证)upstream->执行负载均衡算法->获取到clientconf
	// 获取clientchannel
	localPattern, params := ags.GetLocationPattern(agentChannel, packet)
	location := ags.locationHandle(ags, localPattern)
	if location == nil {
		s := "handle localtion error, Pattern:" + localPattern
		logx.Error(s)
		return errors.New(s)
	}

	// 2、通过负载均衡获取client配置
	upstreamId := location.GetUpstreamId()
	logx.Debug("UpstreamId:", upstreamId)
	upsStreams := ags.GetParent().(IService).GetUpstreams()
	ups, found := upsStreams[upstreamId]
	if found {
		context := NewUpstreamContext(agentChannel, packet, params...)
		ups.SelectDstChannel(context)
		f := context.IsOk()
		logx.Info("select ok:", f)
		if f {
			agentChannel.AddAttach(Upstream_Attach_key, ups)
			// agentChannel.AddAttach(Activating_Key, true)
			return nil
		}
	}
	errMs := "select DstChannel error."
	logx.Error(errMs, ups)
	return errors.New(errMs)
}

const Opcode_Key = "opcode"

// locationHandle 获取location，以便确认upstream的处理
// Pattern 匹配路径
// params 任意参数
type LocationHandle func(server IAgServer, pattern string, params ...interface{}) ILocationConf

// defaultLocationHandle 默认LocationHandle，使用随机分配算法
func defaultLocationHandle(server IAgServer, pattern string, params ...interface{}) ILocationConf {
	aglc := defaultLocationConf
	if len(pattern) >= 0 {
		locations := server.GetConf().GetLocationConfs()
		if locations != nil {
			lc, found := locations[pattern]
			if found {
				aglc = lc
			}
		}
		if aglc == nil {
			logx.Warnf("Pattern:%v, locationConf is nil", pattern)
		}
	}
	logx.Debugf("Pattern:%v, locationConf:%v", pattern, aglc.GetUpstreamId())
	return aglc
}

var defaultLocationConf ILocationConf = NewLocationConf("", "", nil)

