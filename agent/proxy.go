/*
 * Author:slive
 * DATE:2020/8/16
 */
package agent

import (
	"github.com/Slive/gsfly/channel"
	logx "github.com/Slive/gsfly/logger"
	"github.com/Slive/gsfly/socket"
	"github.com/emirpasic/gods/maps/hashmap"
)

type IProxy interface {
	IUpstream
}

// Proxy 通用的代理一对一代理方式，即agent端和dst端是一对一关系
type Proxy struct {
	Upstream
	ProxyConf IProxyConf
	// 记录agent代理端和dst端channelId映射关系
	AgentMapperDstCh *hashmap.Map
}

func NewProxy(parent interface{}, proxyConf IProxyConf, transfer IExtension) *Proxy {
	p := &Proxy{}
	p.Upstream = *NewUpstream(parent, proxyConf, transfer)
	p.ProxyConf = proxyConf
	p.AgentMapperDstCh = hashmap.New()
	return p
}

func (proxy *Proxy) InitChannelPeer(ctx *UpstreamContext) {
	lbsCtx := NewLoadBalanceContext(nil, proxy, ctx.Channel)
	lbhandle := localBalanceHandles[proxy.ProxyConf.GetLoadBalanceType()]
	// TODO 负载均衡策略
	lbhandle(lbsCtx)
	dstClientConf := lbsCtx.Result

	// 3、代理到目标
	var dstCh channel.IChannel
	var params map[string]interface{}
	ps := ctx.Params
	if len(ps) > 0 {
		params = ps[0].(map[string]interface{})
	}
	logx.Info("select params:", params)
	handle := channel.NewDefChHandle(proxy.onDstChannelMsgHandler)
	handle.SetOnActiveHandler(proxy.onDstChannelActiveHandler)
	handle.SetOnInActiveHandler(proxy.onDstChannelInActiveHandler)
	clientStrap := socket.NewClientConn(proxy, dstClientConf, handle, params)
	err := clientStrap.Dial()
	agentCh := ctx.Channel
	agentChId := agentCh.GetId()
	if err != nil {
		logx.Error("dialws error, agentChId:" + agentChId)
		return
	}

	// 拨号成功，记录
	dstCh = clientStrap.GetChannel()
	dstChId := dstCh.GetId()
	proxy.AgentMapperDstCh.Put(agentChId, dstChId)
	// dstChId作为主键
	proxy.GetChannelPeerMap().Put(dstChId, NewChannelPeer(agentCh, dstCh))

	// 记录dstchannel到pool中
	proxy.GetDstChannelMap().Put(dstChId, dstCh)
	ctx.SetRet(dstCh)
}

// GetChannelPeer 通过UpstreamContext获取到对应的channelpeer
func (proxy *Proxy) GetChannelPeer(ctx *UpstreamContext) IChannelPeer {
	var dstChId interface{}
	channel := ctx.GetChannel()
	chId := channel.GetId()
	if ctx.IsAgent() {
		// 代理端，先获取dstChId
		ret, found := proxy.AgentMapperDstCh.Get(chId)
		if !found {
			logx.Warn("get dstChId is nil, agentChId:", chId)
			return nil
		} else {
			dstChId = ret
		}
	} else {
		dstChId = channel.GetId()
	}

	ret, found := proxy.ChannelPeerMap.Get(dstChId)
	if found {
		return ret.(IChannelPeer)
	}
	logx.Warn("get channelpeer is nil, chId:", chId)
	return nil
}

// onDstChannelMsgHandler upstream的客户端channel收到消息后的处理，直接会写到server对应的客户端channel
func (proxy *Proxy) onDstChannelMsgHandler(ctx channel.IChHandlerContext) {
	packet := ctx.GetPacket()
	upsCtx := NewUpstreamContext(packet.GetChannel(), packet, false)
	proxy.QueryAgentChannel(upsCtx)
	agentCh, found := upsCtx.GetRet(), upsCtx.IsOk()
	if found {
		proxy.GetTransfer().Transfer(ctx, agentCh)
		return
	}
	logx.Warn("unknown dst Transfer.")
}

// onDstChannelActiveHandler 当dstchannel关闭时，触发agentchannel关闭
func (proxy *Proxy) onDstChannelActiveHandler(ctx channel.IChHandlerContext) {
	// TODO
}

// onDstChannelInActiveHandler 当dstchannel关闭时，触发agentchannel关闭
func (proxy *Proxy) onDstChannelInActiveHandler(ctx channel.IChHandlerContext) {
	dstChannel := ctx.GetChannel()
	dstChId := dstChannel.GetId()
	defer func() {
		ret := recover()
		logx.Infof("finish to onDstChannelInActiveHandler, chId:%v, ret:%v", dstChId, ret)
	}()

	// 当clientchannel关闭时，触发serverchannel关闭
	logx.Info("start to onDstChannelInActiveHandler, chId:", dstChId)
	proxy.ReleaseByDstChannel(dstChannel)
}

// ReleaseByAgentChannel 当agent端channel释放资源后调用该方法，清除agent端相关的channel的记录
// 因为和dst端channel是一对一对应关系，所以需要释放dst端的channel记录和资源
func (proxy *Proxy) ReleaseByAgentChannel(agentChannel channel.IChannel) {
	agentChId := agentChannel.GetId()
	dstCh, found := proxy.AgentMapperDstCh.Get(agentChId)
	logx.Infof("dstCh found:%v, agentChId:%v", found, agentChId)
	if found {
		// 清除dstchannel相关记录
		proxy.AgentMapperDstCh.Remove(agentChId)
		chPeer, ok := dstCh.(IChannelPeer)
		if ok {
			dstChannel := chPeer.GetDstChannel()
			dstChId := dstChannel.GetId()
			proxy.GetDstChannelMap().Remove(dstChId)
			// 释放dstchannel资源
			dstChannel.Stop()
		}
	}
}

// ReleaseByDstChannel 当dst端channel释放资源后调用该方法，清除dst端相关的channel的记录
// 因为和agent端channel是一对一对应关系，所以需要释放agent端的channel记录和资源
func (proxy *Proxy) ReleaseByDstChannel(dstChannel channel.IChannel) {
	dstChId := dstChannel.GetId()
	chPeer, found := proxy.ChannelPeerMap.Get(dstChId)
	logx.Infof("agentch found:%v, dstChId:%v", found, dstChId)
	if found {
		proxy.GetChannelPeerMap().Remove(dstChId)
		proxy.GetDstChannelMap().Remove(dstChId)
		agentCh := chPeer.(IChannelPeer).GetAgentChannel()
		agentCh.Stop()
	}
}

func (proxy *Proxy) QueryDstChannel(ctx *UpstreamContext) {
	InnerQueryDstChannel(proxy, ctx)
}

func (proxy *Proxy) QueryAgentChannel(ctx *UpstreamContext) {
	InnerQueryAgentChannel(proxy, ctx)
}
