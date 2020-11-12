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

	// ProxyConf 代理配置
	ProxyConf IProxyConf

	// 记录agent代理端和dst端channelId映射关系
	agentMapperDstCh *hashmap.Map
}

func NewProxy(parent interface{}, proxyConf IProxyConf, transfer IExtension) *Proxy {
	p := &Proxy{}
	p.Upstream = *NewUpstream(parent, proxyConf, transfer)
	p.ProxyConf = proxyConf
	p.agentMapperDstCh = hashmap.New()
	return p
}

func (proxy *Proxy) InitChannelPeer(agentCtx channel.IChHandleContext, params map[string]interface{}) {
	lbsCtx := NewLoadBalanceContext(nil, proxy, agentCtx.GetChannel())
	lbhandle := localBalanceHandles[proxy.ProxyConf.GetLoadBalanceType()]
	// TODO 负载均衡策略
	lbhandle(lbsCtx)
	dstClientConf := lbsCtx.Result

	// 3、代理到目标
	var dstCh channel.IChannel
	logx.Info("select params:", params)

	// 初始化DstClientConn
	handle := channel.NewDefChHandle(proxy.onDstChannelReadHandle)
	handle.SetOnActive(proxy.onDstChannelActiveHandle)
	handle.SetOnInActive(proxy.onDstChannelInActiveHandle)
	clientConn := socket.NewClientConn(proxy, dstClientConf, handle, params)
	err := clientConn.Dial()
	agentCh := agentCtx.GetChannel()
	agentChId := agentCh.GetId()
	if err != nil {
		logx.Error("dialws error, agentChId:" + agentChId)
		return
	}

	// 拨号成功，记录
	dstCh = clientConn.GetChannel()
	dstChId := dstCh.GetId()
	// agentChId和dstChId关系
	proxy.agentMapperDstCh.Put(agentChId, dstChId)
	// dstChId作为主键
	proxy.GetChannelPeers().Put(dstChId, NewChannelPeer(agentCh, dstCh))

	// 记录dstchannel到pool中
	proxy.GetDstChannels().Put(dstChId, dstCh)
	agentCtx.SetRet(dstCh)
	logx.Info("fininsh initChannelPeer, agentChId:{}, dstChId:{}", agentChId, dstChId)
}

// GetChannelPeer 通过UpstreamContext获取到对应的channelpeer
func (proxy *Proxy) GetChannelPeer(ctx channel.IChHandleContext, isAgent bool) IChannelPeer {
	var dstChId interface{}
	channel := ctx.GetChannel()
	chId := channel.GetId()
	if isAgent {
		// 代理端，先获取dstChId
		ret, found := proxy.agentMapperDstCh.Get(chId)
		if !found {
			logx.Warn("get dstChId is nil, agentChId:", chId)
			return nil
		} else {
			dstChId = ret
		}
	} else {
		dstChId = channel.GetId()
	}

	ret, found := proxy.channelPeers.Get(dstChId)
	if found {
		return ret.(IChannelPeer)
	}
	logx.Warn("get channelpeer is nil, chId:", chId)
	return nil
}

// onDstChannelReadHandle upstream的客户端channel收到消息后的处理，直接会写到server对应的客户端channel
func (proxy *Proxy) onDstChannelReadHandle(dstCtx channel.IChHandleContext) {
	proxy.QueryAgentChannel(dstCtx)
	agentCh := dstCtx.GetRet()
	if agentCh != nil {
		proxy.extension.Transfer(dstCtx, agentCh.(channel.IChannel))
		return
	}
	logx.Warn("unknown dst Transfer.")
}

// onDstChannelActiveHandle 当dstchannel关闭时，触发agentchannel关闭
func (proxy *Proxy) onDstChannelActiveHandle(ctx channel.IChHandleContext) {
	// TODO
}

// onDstChannelInActiveHandle 当dstchannel关闭时，触发agentchannel关闭
func (proxy *Proxy) onDstChannelInActiveHandle(ctx channel.IChHandleContext) {
	dstChannel := ctx.GetChannel()
	dstChId := dstChannel.GetId()
	defer func() {
		ret := recover()
		logx.Infof("finish to onDstChannelInActiveHandle, chId:%v, ret:%v", dstChId, ret)
	}()

	// 当clientchannel关闭时，触发serverchannel关闭
	logx.Info("start to onDstChannelInActiveHandle, chId:", dstChId)
	proxy.ReleaseOnDstChannel(ctx)
}

// ReleaseOnAgentChannel 当agent端channel释放资源后调用该方法，清除agent端相关的channel的记录
// 因为和dst端channel是一对一对应关系，所以需要释放dst端的channel记录和资源
func (proxy *Proxy) ReleaseOnAgentChannel(agentCtx channel.IChHandleContext) {
	agentChId := agentCtx.GetChannel().GetId()
	dstCh, found := proxy.agentMapperDstCh.Get(agentChId)
	logx.Infof("dstCh found:%v, agentChId:%v", found, agentChId)
	if found {
		// 清除dstchannel相关记录
		proxy.agentMapperDstCh.Remove(agentChId)
		chPeer, ok := dstCh.(IChannelPeer)
		if ok {
			dstChannel := chPeer.GetDstChannel()
			dstChId := dstChannel.GetId()
			proxy.GetDstChannels().Remove(dstChId)
			// 释放dstchannel资源
			dstChannel.Stop()
			dstChannel.GetParent()
		}
	}
}

// ReleaseOnDstChannel 当dst端channel释放资源后调用该方法，清除dst端相关的channel的记录
// 因为和agent端channel是一对一对应关系，所以需要释放agent端的channel记录和资源
func (proxy *Proxy) ReleaseOnDstChannel(dstCtx channel.IChHandleContext) {
	dstChId := dstCtx.GetChannel().GetId()
	chPeer, found := proxy.channelPeers.Get(dstChId)
	logx.Infof("agentch found:%v, dstChId:%v", found, dstChId)
	if found {
		proxy.GetChannelPeers().Remove(dstChId)
		proxy.GetDstChannels().Remove(dstChId)
		agentCh := chPeer.(IChannelPeer).GetAgentChannel()
		agentCh.Stop()
	}
}

func (proxy *Proxy) QueryDstChannel(ctx channel.IChHandleContext) {
	InnerQueryDstChannel(proxy, ctx)
}

func (proxy *Proxy) QueryAgentChannel(ctx channel.IChHandleContext) {
	InnerQueryAgentChannel(proxy, ctx)
}
