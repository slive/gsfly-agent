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
)

// IMsgHandler 处理消息的接口
type IMsgHandler interface {
	Handle(ctx gch.IChHandleContext)
}

type IAgServer interface {
	socket.IServerListener

	// AddMsgHandler 添加处理消息handler，TODO 做排序？
	AddMsgHandler(handler ...IMsgHandler)

	GetMsgHandlers() []IMsgHandler

	ClearMsgHandlers()

	// GetExtension 扩展实现
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
	s.msgHandlers = []IMsgHandler{}

	// 初始化channel相关handler
	handle := gch.NewDefChHandle(s.onAgentChannelReadHandle)
	handle.SetOnActive(s.onAgentChannelActiveHandle)
	handle.SetOnInActive(s.onAgentChannelInActiveHandle)

	// 扩展点
	if extension == nil {
		s.extension = NewExtension()
	} else {
		s.extension = extension
	}

	sConf := serverConf.GetServerConf()
	s.ServerListener = *socket.NewServerListener(parent, sConf, handle)
	return s
}

func (server *AgServer) Listen() error {
	err := server.extension.BeforeServerListen(server)
	defer func() {
		if err == nil {
			// 成功后的操作
			server.extension.AfterServerListen(server)
		}
	}()
	if err == nil {
		err = server.ServerListener.Listen()
		if err == nil {
			server.locationHandle = defaultLocationHandle
		}
	}
	return err
}

func (server *AgServer) GetExtension() IExtension {
	return server.extension
}

func (server *AgServer) GetConf() socket.IServerConf {
	return server.serverConf
}

// AddMsgHandler 添加处理消息handler，按添加顺序先后次序执行
func (server *AgServer) AddMsgHandler(handler ...IMsgHandler) {
	server.msgHandlers = append(server.msgHandlers, handler...)
}

func (server *AgServer) GetMsgHandlers() []IMsgHandler {
	return server.msgHandlers
}

func (server *AgServer) ClearMsgHandlers() {
	server.msgHandlers = []IMsgHandler{}
}

const (
	Upstream_Attach_key = "upstream"
)

// onAgentChannelActiveHandle 当agentChannel注册时，路由dstClientChannel等操作
func (ags *AgServer) onAgentChannelActiveHandle(ctx gch.IChHandleContext) {
	chId := ctx.GetChannel().GetId()
	extension := ags.extension
	err := extension.BeforeAgentChannelActive(ctx)
	if err != nil {
		logx.Error("check active, chId:%v, error:%v", chId, err)
		gch.NotifyErrorHandle(ctx, err, gch.ERR_ACTIVE)
		return
	}
	ags.locationUpstream(ctx)
	err = ctx.GetError()
	if err != nil {
		logx.Error("location chId:%v, error:%v", chId, err)
		gch.NotifyErrorHandle(ctx, err, gch.ERR_ACTIVE)
	} else {
		// 最后的处理
		extension.AfterAgentChannelActive(ctx)
	}
}

// onAgentChannelInActiveHandle 当agentChannel关闭时，触发dstClientChannel关闭
func (ags *AgServer) onAgentChannelInActiveHandle(ctx gch.IChHandleContext) {
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
			proxy.ReleaseOnAgentChannel(ctx)
		}
	}
}

func (ags *AgServer) onAgentChannelReadHandle(handlerCtx gch.IChHandleContext) {
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
		ups.QueryDstChannel(handlerCtx)
		dstCh := handlerCtx.GetRet()
		if dstCh != nil {
			ags.GetExtension().Transfer(handlerCtx, dstCh.(gch.IChannel))
			return
		}
	}
	logx.Warn("unknown agent Transfer.")
}

const default_localPattern = ""

// locationUpstream 定位到location
func (ags *AgServer) locationUpstream(agentCtx gch.IChHandleContext) {
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

	// 通过GetLocationPattern获取到对应的urlpattern，从而获取到upsetream
	agentChannel := agentCtx.GetChannel()
	localPattern, params := ags.GetExtension().GetLocationPattern(agentCtx)
	location := ags.locationHandle(ags, localPattern)
	if location == nil {
		s := "handle localtion error, Pattern:" + localPattern
		logx.Error(s)
		agentCtx.SetError(common.NewError2(gch.ERR_MSG, s))
		return
	}

	// 2、通过负载均衡获取client配置
	upstreamId := location.GetUpstreamId()
	logx.Debug("upstreamId:", upstreamId)
	upsStreams := ags.GetParent().(IService).GetUpstreams()
	ups, found := upsStreams[upstreamId]
	if found {
		// 第一次获取到upstream，要构建channelPeer，然后对agentChannel和dstChannel进行关联
		ups.InitChannelPeer(agentCtx, params)
		ret := agentCtx.GetRet()
		isOk := (ret != nil)
		logx.Info("select ret:", isOk)
		if isOk {
			// TODO 一个agent可能有多个upstream情况
			agentChannel.AddAttach(Upstream_Attach_key, ups)
			return
		}
	}

	errMs := "select dstchannel error."
	logx.Error(errMs, ups)
	agentCtx.SetError(common.NewError2(gch.ERR_MSG, errMs))
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
