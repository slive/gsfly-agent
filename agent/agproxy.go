/*
 * Author:slive
 * DATE:2020/8/16
 */
package agent

import (
	"github.com/Slive/gsfly/bootstrap"
	"github.com/Slive/gsfly/channel"
	logx "github.com/Slive/gsfly/logger"
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

func NewProxy(parent interface{}, proxyConf IProxyConf) *Proxy {
	p := &Proxy{}
	p.Upstream = *NewUpstream(parent, proxyConf)
	p.ProxyConf = proxyConf
	p.AgentMapperDstCh = hashmap.New()
	return p
}

func (proxy *Proxy) CreateChannelPeer(ctx *UpstreamContext) {
	lbsCtx := NewLbContext(nil, proxy, ctx.Channel)
	lbhandle := localBalanceHandles[proxy.ProxyConf.GetLoadBalanceType()]
	// TODO 负载均衡策略
	lbhandle(lbsCtx)
	dstClientConf := lbsCtx.DstClientConf

	// 3、代理到目标
	var dstCh channel.IChannel
	params := ctx.Params[0].(map[string]interface{})
	logx.Info("select params:", params)
	agentCh := ctx.Channel
	var clientStrap bootstrap.IClientStrap
	clientPro := dstClientConf.GetNetwork()
	switch clientPro {
	case channel.NETWORK_WS, channel.NETWORK_HTTPX:
		handle := channel.NewDefChHandle(proxy.OnDstChannelMsgHandle)
		wsClientConf := dstClientConf.(*bootstrap.WsClientConf)
		handle.SetOnRegisteredHandle(onDstChannelRegHandle(clientPro))
		handle.SetOnUnRegisteredHandle(proxy.OnDstChannelUnRegHandle)
		clientStrap = bootstrap.NewWsClientStrap(proxy, wsClientConf, handle, params)
	case channel.NETWORK_HTTP:
		break
	case channel.NETWORK_KWS00:
		kwsClientConf := dstClientConf.(*bootstrap.Kws00ClientConf)
		clientStrap = bootstrap.NewKws00ClientStrap(proxy, kwsClientConf, proxy.OnDstChannelMsgHandle,
			onDstChannelRegHandle(clientPro), proxy.OnDstChannelUnRegHandle, params)
	case channel.NETWORK_KWS01:
		break
	case channel.NETWORK_TCP:
		break
	case channel.NETWORK_UDP:
		break
	case channel.NETWORK_KCP:
		break
	default:
		// channel.NETWORK_WS
	}

	if clientStrap != nil {
		handle := clientStrap.GetChHandle()
		handle.SetOnStopHandle(proxy.OnDstChannelStopHandle)
		err := clientStrap.Start()
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
		proxy.ChannelPeerMap.Put(dstChId, NewChannelPeer(agentCh, dstCh))

		// 记录dstchannel到pool中
		proxy.GetDstChannelPool().Put(dstChId, dstCh)
	}
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

// OnDstChannelMsgHandle upstream的客户端channel收到消息后的处理，直接会写到server对应的客户端channel
func (proxy *Proxy) OnDstChannelMsgHandle(packet channel.IPacket) error {
	upsCtx := NewUpstreamContext(packet.GetChannel(), packet, false)
	proxy.QueryAgentChannel(upsCtx)
	agentChannel, found := upsCtx.GetRet(), upsCtx.IsOk()
	if found {
		ProxyWrite(packet, agentChannel)
		return nil
		// protocol := agentChannel.GetConf().GetNetwork()
		// switch protocol {
		// case channel.NETWORK_WS, channel.NETWORK_HTTPX:
		// 	// 回写，区分第一次，最后一次等？
		// 	wsChannel := agentChannel.(*httpx.WsChannel)
		// 	srcPacket := wsChannel.NewPacket().(*httpx.WsPacket)
		// 	srcPacket.SetData(packet.GetData())
		// 	srcPacket.MsgType = websocket.TextMessage
		// 	agentChannel.Write(srcPacket)
		// 	return nil
		// case channel.NETWORK_HTTP:
		// case channel.NETWORK_KWS00:
		// 	// 回写，区分第一次，最后一次等？
		// 	opcode := gkcp.OPCODE_TEXT_SIGNALLING
		// 	oc := agentChannel.GetAttach(Opcode_Key)
		// 	if oc != nil {
		// 		cpc, ok := oc.(uint16)
		// 		if ok {
		// 			opcode = cpc
		// 		}
		// 	}
		// 	frame := gkcp.NewOutputFrame(opcode, packet.GetData())
		// 	srcPacket := agentChannel.NewPacket()
		// 	srcPacket.SetData(frame.GetKcpData())
		// 	agentChannel.Write(srcPacket)
		// 	return nil
		// case channel.NETWORK_KWS01:
		// 	break
		// case channel.NETWORK_TCP:
		// 	break
		// case channel.NETWORK_UDP:
		// 	break
		// case channel.NETWORK_KCP:
		// 	break
		// default:
		// }
	}
	logx.Warn("unknown dst ProxyWrite.")
	return nil
}

func onDstChannelRegHandle(agentNetwork channel.Network) func(dstChannel channel.IChannel, packet channel.IPacket) error {
	return func(dstChannel channel.IChannel, packet channel.IPacket) error {
		if agentNetwork == channel.NETWORK_KWS00 {
			dstChannel.AddAttach(Activating_Key, true)
		}
		logx.Info("register success:", dstChannel.GetId())
		return nil
	}
}

// onDstChannelUnRegHandle 当dstchannel取消注册后时，触发释放资源
func (proxy *Proxy) OnDstChannelUnRegHandle(dstChannel channel.IChannel, packet channel.IPacket) error {
	dstChId := dstChannel.GetId()
	defer func() {
		ret := recover()
		logx.Infof("finish to onDstChannelUnRegHandle, chId:%v, ret:%v", dstChId, ret)
	}()

	// 当clientchannel关闭时，触发serverchannel关闭
	logx.Info("start to onDstChannelUnRegHandle, chId:", dstChId)
	return nil
}

// OnDstChannelStopHandle 当dstchannel关闭时，触发agentchannel关闭
func (proxy *Proxy) OnDstChannelStopHandle(dstChannel channel.IChannel) error {
	dstChId := dstChannel.GetId()
	defer func() {
		ret := recover()
		logx.Infof("finish to OnDstChannelStopHandle, chId:%v, ret:%v", dstChId, ret)
	}()

	// 当clientchannel关闭时，触发serverchannel关闭
	logx.Info("start to OnDstChannelStopHandle, chId:", dstChId)
	proxy.ReleaseByDstChannel(dstChannel)
	return nil
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
			proxy.GetDstChannelPool().Remove(dstChId)
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
		proxy.ChannelPeerMap.Remove(dstChId)
		proxy.GetDstChannelPool().Remove(dstChId)
		agentCh := chPeer.(IChannelPeer).GetAgentChannel()
		agentCh.Stop()
	}
}
