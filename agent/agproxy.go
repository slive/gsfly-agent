/*
 * Author:slive
 * DATE:2020/8/16
 */
package agent

import (
	"gsfly/bootstrap"
	"gsfly/channel"
	logx "gsfly/logger"
)

type IProxy interface {
	IUpstream
}

type Proxy struct {
	Upstream

	LoadBalance LoadBalance
}

func NewProxy(parent interface{}, proxyConf IProxyConf) *Proxy {
	p := &Proxy{}
	p.Upstream = *NewUpstream(parent, proxyConf)
	return p
}

func (proxy *Proxy) SelectDstChannel(ctx *UpstreamContext) {
	lbsCtx := NewLbContext(nil, proxy, ctx.Channel)
	lbhandle := localBalanceHandles[proxy.LoadBalance.GetType()]
	lbhandle(lbsCtx)
	clientConf := lbsCtx.DstClientConf

	// 3、代理到目标
	var dstCh channel.IChannel
	params := ctx.Params[0].(map[string]interface{})
	logx.Info("select params:", params)
	agentCh := ctx.Channel
	var clientStrap bootstrap.IClientStrap
	clientPro := clientConf.GetProtocol()
	switch clientPro {
	case channel.PROTOCOL_WS, channel.PROTOCOL_HTTPX:
		handle := channel.NewDefChHandle(proxy.OnDstChannelMsgHandle)
		wsClientConf := clientConf.(*bootstrap.WsClientConf)
		handle.SetOnRegisteredHandle(proxy.OnDstChannelRetHandle)
		clientStrap = bootstrap.NewWsClientStrap(proxy, wsClientConf, handle, params)
	case channel.PROTOCOL_HTTP:
		break
	case channel.PROTOCOL_KWS00:
		kwsClientConf := clientConf.(*bootstrap.Kws00ClientConf)
		clientStrap = bootstrap.NewKws00ClientStrap(proxy, kwsClientConf, proxy.OnDstChannelMsgHandle,
			proxy.OnDstChannelRetHandle, nil, nil)
	case channel.PROTOCOL_KWS01:
		break
	case channel.PROTOCOL_TCP:
		break
	case channel.PROTOCOL_UDP:
		break
	case channel.PROTOCOL_KCP:
		break
	default:
		// channel.PROTOCOL_WS
	}

	found := (clientStrap != nil)
	if found {
		handle := clientStrap.GetChHandle()
		handle.SetOnStopHandle(proxy.OnDstChannelStopHandle)
		err := clientStrap.Start()
		if err != nil {
			logx.Error("dialws error, agentChId:" + agentCh.GetId())
			return
		}

		// 拨号成功，记录
		dstCh = clientStrap.GetChannel()
		proxy.GetAgentChannelMap().Put(dstCh.GetId(), agentCh)
		proxy.GetDstChannelMap().Put(agentCh.GetId(), dstCh)
		proxy.GetDstChannelPool().Put(dstCh.GetId(), dstCh)
	}
	ctx.SetRet(dstCh)
}

// OnDstChannelMsgHandle upstream的客户端channel收到消息后的处理，直接会写到server对应的客户端channel
func (proxy *Proxy) OnDstChannelMsgHandle(packet channel.IPacket) error {
	upsCtx := NewUpstreamContext(packet.GetChannel(), packet)
	proxy.QueryAgentChannel(upsCtx)
	agentChannel, found := upsCtx.GetRet(), upsCtx.IsOk()
	if found {
		ProxyWrite(packet, agentChannel)
		return nil
		// protocol := agentChannel.GetConf().GetProtocol()
		// switch protocol {
		// case channel.PROTOCOL_WS, channel.PROTOCOL_HTTPX:
		// 	// 回写，区分第一次，最后一次等？
		// 	wsChannel := agentChannel.(*httpx.WsChannel)
		// 	srcPacket := wsChannel.NewPacket().(*httpx.WsPacket)
		// 	srcPacket.SetData(packet.GetData())
		// 	srcPacket.MsgType = websocket.TextMessage
		// 	agentChannel.Write(srcPacket)
		// 	return nil
		// case channel.PROTOCOL_HTTP:
		// case channel.PROTOCOL_KWS00:
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
		// case channel.PROTOCOL_KWS01:
		// 	break
		// case channel.PROTOCOL_TCP:
		// 	break
		// case channel.PROTOCOL_UDP:
		// 	break
		// case channel.PROTOCOL_KCP:
		// 	break
		// default:
		// }
	}
	logx.Warn("unknown dst ProxyWrite.")
	return nil
	return nil
}

func (proxy *Proxy) OnDstChannelRetHandle(dstChannel channel.IChannel, packet channel.IPacket) error {
	logx.Info("register success:", dstChannel.GetId())
	dstChannel.AddAttach(Activating_Key, true)
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
	proxy.ClearDstChannel(dstChannel)

	return nil
}

func (proxy *Proxy) TakeChannnelKey(ctx *UpstreamContext) (routeId string) {
	return ctx.Channel.GetId()
}

func (proxy *Proxy) QueryDstChannel(ctx *UpstreamContext) {
	InnerQueryDstChannel(proxy, ctx)
}

func (proxy *Proxy) QueryAgentChannel(ctx *UpstreamContext) {
	InnerQueryAgentChannel(proxy, ctx)
}

func (proxy *Proxy) ClearAgentChannel(agentChannel channel.IChannel) {
	dstCh, found := proxy.GetDstChannelMap().Get(agentChannel.GetId())
	if found {
		dstChannel, ok := dstCh.(channel.Channel)
		if ok {
			dstChId := dstChannel.GetId()
			dstChannel.Stop()
			proxy.GetAgentChannelMap().Remove(dstChId)
			proxy.GetDstChannelPool().Remove(dstChId)
		}
		proxy.GetDstChannelMap().Remove(agentChannel.GetId())
	}
}

func (proxy *Proxy) ClearDstChannel(dstChannel channel.IChannel) {
	dstChId := dstChannel.GetId()
	agentCh, found := proxy.GetAgentChannelMap().Get(dstChId)
	if found {
		agetnChannel, ok := agentCh.(channel.Channel)
		if ok {
			agetnChannel.Stop()
			proxy.GetDstChannelMap().Remove(agetnChannel.GetId())
		}
		proxy.GetAgentChannelMap().Remove(agetnChannel.GetId())
		proxy.GetDstChannelPool().Remove(dstChId)
	}
}
