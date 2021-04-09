/*
 * 扩展实现，主要实现如：
 *  1、初始化Upstream
 *  2、agent(dstClient)转换转发功能
 *  3、定位匹配功能
 * Author:slive
 * DATE:2020/10/14
 */
package agent

import (
	gch "github.com/slive/gsfly/channel"
	"github.com/slive/gsfly/channel/tcpx"
	"github.com/slive/gsfly/common"
	logx "github.com/slive/gsfly/logger"
	"github.com/gorilla/websocket"
)

// IExtension 代理扩展，实现相关联接口
type IExtension interface {
	common.IParent

	// Transfer 代理的转发消息操作，可能是agentChannel收到消息后转发到dstChannel，也可能是dstChannel收到消息后转发到agentChannel
	// fromCtx 从哪里来的结束的消息
	// toChannel 到目标channel发送
	Transfer(fromCtx gch.IChHandleContext, toChannel gch.IChannel)

	// BeforeAgentChannelActive 当agentChannel激活时的操作，检测出错，则停止后面的操作
	BeforeAgentChannelActive(ctx gch.IChHandleContext) error

	// AfterAgentChannelActive 当agentChannel激活的操作成功后，完成后的操作
	AfterAgentChannelActive(ctx gch.IChHandleContext)

	// GetLocationPattern 获取location匹配路径和对应的参数，然后可通过localPattern查找到对应已初始化的IUpstream
	GetLocationPattern(ctx gch.IChHandleContext) (localPattern string, params map[string]interface{})

	// CreateUpstream 实现不同的Upstream，如自定义的upstream
	CreateUpstream(upsConf IUpstreamConf) IUpstream

	// BeforeServerListen 在ServerListen前操作，如果报错，则无法进行ServerListen操作
	BeforeServerListen(server IAgServer) error

	// AfterServerListen 在ServerListen后的操作
	AfterServerListen(server IAgServer)

	// GetAgentMsgHandlers 获取代理agentchannel的消息处理，按顺序提供
	GetAgentMsgHandlers() []IMsgHandler

	// GetExtConf 获取扩展参数
	GetExtConf() map[string]string

	// SetExtConf 设置扩展参数
	SetExtConf(extConf map[string]string)
}

// Extension 代理扩展实现，所有代理可实现功能，基本都在这里实现
type Extension struct {
	common.Parent

	// 消息处理
	msgHandles []IMsgHandler

	// 扩展的配置
	extConf map[string]string
}

// NewExtension 创建代理扩展实现
// msgHandles 消息处理，按顺序处理
func NewExtension(msgHandles ...IMsgHandler) *Extension {
	e := &Extension{}
	e.Parent = *common.NewParent(nil)
	e.msgHandles = msgHandles
	return e
}

// Transfer 代理的转发消息操作，可能是agentChannel收到消息后转发到dstChannel，也可能是dstChannel收到消息后转发到agentChannel
// fromCtx 从哪里来的结束的消息
// toChannel 到目标channel发送
func (e *Extension) Transfer(fromCtx gch.IChHandleContext, toChannel gch.IChannel) {
	fromPacket := fromCtx.GetPacket()
	fromChannel := fromPacket.GetChannel()
	dstPacket := toChannel.NewPacket()
	fromNetwork := fromChannel.GetConf().GetNetwork()
	toNetwork := toChannel.GetConf().GetNetwork()
	// 根据不同的协议类型，转发到不同的目的toChannel
	switch fromNetwork {
	case gch.NETWORK_WS:
		switch toNetwork {
		case gch.NETWORK_WS:
			dstPacket.SetData(fromPacket.GetData())
			// 同样的协议保持包类型一致
			dstPacket.(*tcpx.WsPacket).MsgType = fromPacket.(*tcpx.WsPacket).MsgType
		default:
			dstPacket.SetData(fromPacket.GetData())
		}
	case gch.NETWORK_KCP:
		switch toNetwork {
		case gch.NETWORK_WS:
			dstPacket.SetData(fromPacket.GetData())
			// 默认text方式？，pingpong等方式呢？
			dstPacket.(*tcpx.WsPacket).MsgType = websocket.TextMessage
		default:
			dstPacket.SetData(fromPacket.GetData())
		}
	default:
		dstPacket.SetData(fromPacket.GetData())
	}
	toChannel.Write(dstPacket)
}

func (e *Extension) GetLocationPattern(ctx gch.IChHandleContext) (localPattern string, params map[string]interface{}) {
	agentChannel := ctx.GetChannel()
	protocol := agentChannel.GetConf().GetNetwork()
	localPattern = ""
	params = make(map[string]interface{})
	switch protocol {
	case gch.NETWORK_WS:
		wsChannel := agentChannel.(*tcpx.WsChannel)
		params = wsChannel.GetParams()
		// 1、约定用path来限定路径，全匹配
		localPattern = wsChannel.GetRelativePath()
	default:
		// TODO 待完善
		localPattern = default_localPattern
	}
	logx.Infof("channel params:%v, localPattern:%v", params, localPattern)
	return localPattern, params
}

// CreateUpstream 实现不同的Upstream，如自定义的upstream
func (e *Extension) CreateUpstream(upsConf IUpstreamConf) IUpstream {
	// 不同的upstreamtype进行不同的处理
	upsType := upsConf.GetUpstreamType()
	var ups IUpstream
	if upsType == UPSTREAM_PROXY {
		proxyConf, ok := upsConf.(IProxyConf)
		if ok {
			ups = NewProxy(e.GetParent(), proxyConf, e)
		} else {
			panic("upstream conf is invalid.")
		}
	} else {
		// TODO
		panic("upstream type is invalid.")
	}
	return ups
}

// BeforeServerListen 在ServerListen前操作，如果报错，则无法进行ServerListen操作
func (e *Extension) BeforeServerListen(server IAgServer) error {
	// 空实现
	logx.Info("before serverlistener, nothing...")
	return nil
}

// AfterServerListen 在ServerListen前操作
func (e *Extension) AfterServerListen(server IAgServer) {
	// 空实现
	logx.Info("after serverlistener, nothing...")
}

// BeforeAgentChannelActive 当agentChannel激活时的操作，检测出错，则停止后面的操作
func (e *Extension) BeforeAgentChannelActive(ctx gch.IChHandleContext) error {
	return nil
}

// AfterAgentChannelActive 当agentChannel激活的操作成功后，完成后的操作
func (e *Extension) AfterAgentChannelActive(ctx gch.IChHandleContext){
	// 空实现
}

// GetAgentMsgHandlers 获取agent msg的操作，默认实现
func (e *Extension) GetAgentMsgHandlers() []IMsgHandler {
	return e.msgHandles
}

// GetExtConf 获取扩展参数
func (e *Extension) GetExtConf() map[string]string {
	return e.extConf
}

// SetExtConf 设置扩展参数
func (e *Extension) SetExtConf(extConf map[string]string) {
	e.extConf = extConf
}
