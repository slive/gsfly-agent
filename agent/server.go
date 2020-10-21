/*
 * Author:slive
 * DATE:2020/9/21
 */
package agent

import (
	gch "github.com/Slive/gsfly/channel"
	"github.com/Slive/gsfly/common"
	logx "github.com/Slive/gsfly/logger"
	"github.com/Slive/gsfly/socket"
	"github.com/emirpasic/gods/maps/hashmap"
)

type IMsgHandler interface {
	Handle(ctx gch.IChHandlerContext)
}

type IAgServer interface {
	socket.IServerListener

	AddMsgHandler(handler IMsgHandler)

	GetMsgHandlers() []IMsgHandler

	ClearMsgHandlers()

	LocationUpstream(ctx gch.IChHandlerContext)

	GetExtension() IExtension
}

// AgServer 代理服务器
type AgServer struct {
	socket.ServerListener

	serverConf IAgServerConf

	// 处理location选择
	locationHandle LocationHandle

	// 处理消息hanndler
	msgHandlers []IMsgHandler

	extension IExtension
}

func NewAgServer(parent interface{}, serverConf IAgServerConf, extension IExtension) *AgServer {
	s := &AgServer{serverConf: serverConf}
	s.Closed = false
	s.ChannelPool = hashmap.New()
	s.msgHandlers = []IMsgHandler{}

	// 初始化channel相关handler
	handle := gch.NewDefChHandle(s.onAgentChannelHandler)
	handle.SetOnActiveHandler(s.onAgentChannelActiveHandle)
	handle.SetOnInActiveHandler(s.onAgentChannelInActiveHandle)

	// 扩展点
	if extension == nil {
		s.extension = NewExtension()
	} else {
		s.extension = extension
	}

	s.ServerListener = *socket.NewServerListener(parent, serverConf.GetServerConf(), handle)
	return s
}

func (server *AgServer) Listen() error {
	err := server.ServerListener.Listen()
	if err == nil {
		server.locationHandle = defaultLocationHandle
	}
	return err
}

func (server *AgServer) GetExtension() IExtension {
	return server.extension
}

func (server *AgServer) GetConf() socket.IServerConf {
	return server.serverConf
}

func (server *AgServer) AddMsgHandler(handler IMsgHandler) {
	server.msgHandlers = append(server.msgHandlers, handler)
}

func (server *AgServer) GetMsgHandlers() []IMsgHandler {
	return server.msgHandlers
}

func (server *AgServer) ClearMsgHandlers() {
	server.msgHandlers = []IMsgHandler{}
}

const (
	Upstream_Attach_key = "upstream"
	path_key            = "path"
)

func (ags *AgServer) onAgentChannelActiveHandle(ctx gch.IChHandlerContext) {
	ags.LocationUpstream(ctx)
	err := ctx.GetError()
	if err != nil {
		gch.NotifyErrorHandle(ctx, err, gch.ERR_ACTIVE)
	}
}

// onAgentChannelInActiveHandle 当serverChannel关闭时，触发clientchannel关闭
func (ags *AgServer) onAgentChannelInActiveHandle(ctx gch.IChHandlerContext) {
	agentChannel := ctx.GetChannel()
	agentChId := agentChannel.GetId()
	defer func() {
		ret := recover()
		logx.Infof("finish to onAgentChannelInActiveHandle, chId:%v, ret:%v", agentChId, ret)
	}()
	logx.Info("start to onAgentChannelInActiveHandle, chId:", agentChId)
	ups := agentChannel.GetAttach(Upstream_Attach_key)
	if ups != nil {
		proxy, ok := ups.(IProxy)
		if ok {
			proxy.ReleaseByAgentChannel(agentChannel)
		}
	}
}

func (ags *AgServer) onAgentChannelHandler(handlerCtx gch.IChHandlerContext) {
	packet := handlerCtx.GetPacket()
	// 先处理注册的msghandler
	handlers := ags.msgHandlers
	if handlers != nil {
		for _, handler := range handlers {
			handler.Handle(handlerCtx)
			if packet.IsRelease() {
				return
			}
		}
	}

	agentChannel := packet.GetChannel()
	ups, found := agentChannel.GetAttach(Upstream_Attach_key).(IUpstream)
	if found {
		upCtx := NewUpstreamContext(agentChannel, packet, true)
		ups.QueryDstChannel(upCtx)
		dstCh, found := upCtx.GetRet(), upCtx.IsOk()
		if found {
			ags.GetExtension().Transfer(handlerCtx, dstCh)
			return
		}
	}
	logx.Warn("unknown agent Transfer.")
}

const default_localPattern = ""

func (ags *AgServer) LocationUpstream(ctx gch.IChHandlerContext) {
	// TODO filter的处理
	// FilterConfs := ags.GetServerConf().GetServerConf().GetFilterConfs()

	// 步骤：
	// 先获取(可先认证)location->获取(可先认证)upstream->执行负载均衡算法->获取到clientconf
	// 获取clientchannel
	defer func() {
		ret := recover()
		if ret != nil {
			logx.Error("location upstream error:", ret)
		}
	}()
	agentChannel := ctx.GetChannel()
	localPattern, params := ags.GetExtension().GetLocationPattern(ctx)
	location := ags.locationHandle(ags, localPattern)
	if location == nil {
		s := "handle localtion error, Pattern:" + localPattern
		logx.Error(s)
		ctx.SetError(common.NewError2(gch.ERR_MSG, s))
		return
	}

	// 2、通过负载均衡获取client配置
	upstreamId := location.GetUpstreamId()
	logx.Debug("upstreamId:", upstreamId)
	upsStreams := ags.GetParent().(IService).GetUpstreams()
	ups, found := upsStreams[upstreamId]
	if found {
		context := NewUpstreamContext(agentChannel, ctx.GetPacket(), true, params...)
		ups.InitChannelPeer(context)
		f := context.IsOk()
		logx.Info("select ok:", f)
		if f {
			// TODO 一个agent可能有多个upstream情况
			agentChannel.AddAttach(Upstream_Attach_key, ups)
			return
		}
	}
	errMs := "select DstChannel error."
	logx.Error(errMs, ups)
	ctx.SetError(common.NewError2(gch.ERR_MSG, errMs))
	return
}

// locationHandle 获取location，以便确认upstream的处理
// Pattern 匹配路径
// params 任意参数
type LocationHandle func(server IAgServer, pattern string, params ...interface{}) ILocationConf

// defaultLocationHandle 默认LocationHandle，使用随机分配算法
func defaultLocationHandle(server IAgServer, pattern string, params ...interface{}) ILocationConf {
	aglc := defaultLocationConf
	if len(pattern) >= 0 {
		conf := server.GetConf().(IAgServerConf)
		locations := conf.GetLocationConfs()
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
