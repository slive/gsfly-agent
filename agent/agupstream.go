/*
 * 路由是用户自定义方式实现
 * Author:slive
 * DATE:2020/8/13
 */
package agent

import (
	"errors"
	"github.com/emirpasic/gods/maps/hashmap"
	"gsfly/bootstrap"
	"gsfly/channel"
	gkcp "gsfly/channel/udpx/kcpx"
	"gsfly/common"
	logx "gsfly/logger"
	"gsfly/util"
)

type IUpstream interface {
	common.IParent

	GetConf() IUpstreamConf

	GetDstChannelPool() *hashmap.Map

	GetDstChannelMap() *hashmap.Map

	GetAgentChannelMap() *hashmap.Map

	TakeChannnelKey(ctx *UpstreamContext) (routeId string)

	SelectDstChannel(ctx *UpstreamContext) (retChannel channel.IChannel, found bool)

	QueryDstChannel(ctx *UpstreamContext) (retChannel channel.IChannel, found bool)

	QueryAgentChannel(ctx *UpstreamContext) (retChannel channel.IChannel, found bool)
}

type Upstream struct {
	common.Parent

	Conf IUpstreamConf

	DstChannelPool *hashmap.Map

	AgentChannelMap *hashmap.Map

	DstChannelMap *hashmap.Map
}

func NewUpstream(parent interface{}, upstreamConf IUpstreamConf) *Upstream {
	if upstreamConf == nil {
		errMsg := "upstreamConf id is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	u := &Upstream{
		Conf:            upstreamConf,
		AgentChannelMap: hashmap.New(),
		DstChannelMap:   hashmap.New(),
		DstChannelPool:  hashmap.New(),
	}
	u.SetParent(parent)
	return u
}

func (b Upstream) GetConf() IUpstreamConf {
	return b.Conf
}

// GetDstChannelPool 目标channel池, channelId作为主键
func (b Upstream) GetDstChannelPool() *hashmap.Map {
	return b.GetDstChannelPool()
}

// GetDstChannelMap 记录AgentChannel, 主键为路由的出来的Id
func (b Upstream) GetDstChannelMap() *hashmap.Map {
	return b.DstChannelMap
}

// GetAgentChannelMap 记录AgentChannel, 主键为路由的出来的Id
func (b Upstream) GetAgentChannelMap() *hashmap.Map {
	return b.AgentChannelMap
}

func (b Upstream) TakeChannnelKey(ctx *UpstreamContext) (routeId string) {
	panic("implement me")

}

type Route struct {
	Upstream
}

func (b Route) TakeChannnelKey(ctx *UpstreamContext) (routeId string) {
	panic("implement me")

}

func (b Route) SelectDstChannel(ctx *UpstreamContext) (retChannel channel.IChannel, found bool) {
	routeId := b.TakeChannnelKey(ctx)
	if len(routeId) <= 0 {
		logx.Error("routeId is nil, chId:", ctx.Channel.GetId())
		return nil, false
	}

	maxDstChannelSize := 0
	// TODO 待实现

	if maxDstChannelSize > 0 {
		// 有dstchannel限制
		pool := util.Hashcode(routeId) % maxDstChannelSize
		values := b.DstChannelPool.Values()
		if pool < len(values) {
			retChannel = values[pool].(channel.IChannel)
		}
	}

	hadcreated := retChannel != nil
	if hadcreated {
		b.GetDstChannelMap().Put(routeId, retChannel)
		b.GetAgentChannelMap().Put(routeId, ctx.Channel)
		return retChannel, hadcreated
	} else {
		// TODO 创建...
		panic("wait create dstchannel")
		// b.DstChannelMap.Put(routeId, retChannel)
	}
}

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

func (b Proxy) SelectDstChannel(ctx *UpstreamContext) (retChannel channel.IChannel, found bool) {
	logx.Info("select dst")
	lbsCtx := &LoadBalanceContext{
		Proxy:        &b,
		AgentChannel: ctx.Channel,
	}
	lbhandle := localBalanceHandles[b.LoadBalance.GetType()]
	clientConf := lbhandle(lbsCtx)

	// 3、代理到目标
	params := ctx.Params[0].(map[string]interface{})
	clientPro := clientConf.GetProtocol()
	switch clientPro {
	case channel.PROTOCOL_WS, channel.PROTOCOL_HTTPX:
		hx := ctx.AgServer
		handle := channel.NewDefChHandle(b.onDstChannelMsgHandle)
		// handle.OnStopHandle = b.OnDstChannelStopHandle
		wsClientConf := clientConf.(*bootstrap.WsClientConf)
		wsClientConf.Params = params

		logx.Info("params:", wsClientConf.Params)
		wsClient := bootstrap.NewWsClient(hx, wsClientConf, handle)
		err := wsClient.Start()
		agentCh := ctx.Channel
		if err != nil {
			logx.Error("dialws error, agentChId:" + agentCh.GetId())
			return
		}
		// 拨号成功，记录
		dstCh := wsClient.GetChannel()
		b.GetAgentChannelMap().Put(dstCh.GetId(), agentCh)
		b.GetDstChannelMap().Put(agentCh.GetId(), dstCh)
		return dstCh, true
		break
	case channel.PROTOCOL_HTTP:
		break
	case channel.PROTOCOL_KWS00:
		break
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
	return nil, false
}

// onDstChannelMsgHandle upstream的客户端channel收到消息后的处理，直接会写到server对应的客户端channel
func (hx *Proxy) onDstChannelMsgHandle(packet channel.IPacket) error {
	logx.Info("dst msg")
	context := &UpstreamContext{
		Channel: packet.GetChannel(),
		Packet:  packet,
	}
	agentChannel, found := hx.QueryAgentChannel(context)
	if found {
		// 回写，区分第一次，最后一次等？
		opcode := gkcp.OPCODE_TEXT_SIGNALLING
		oc := agentChannel.GetAttach("opcode")
		if oc != nil {
			cpc, ok := oc.(uint16)
			if ok {
				opcode = cpc
			}
		}
		frame := gkcp.NewOutputFrame(opcode, packet.GetData())
		srcPacket := agentChannel.NewPacket()
		srcPacket.SetData(frame.GetKcpData())
		agentChannel.Write(srcPacket)
		return nil
	} else {
		s := "src channel is nil, dst channel id:" + packet.GetChannel().GetId()
		logx.Error(s)
		return errors.New(s)
	}
}

func (b Proxy) TakeChannnelKey(ctx *UpstreamContext) (routeId string) {
	return ctx.Channel.GetId()
}

func (b Proxy) QueryDstChannel(ctx *UpstreamContext) (retChannel channel.IChannel, found bool) {
	return InnerQueryDstChannel(&b, ctx)
}

func InnerQueryDstChannel(b IUpstream, ctx *UpstreamContext) (retChannel channel.IChannel, found bool) {
	logx.Info("query dst msg")
	routeId := b.TakeChannnelKey(ctx)
	if len(routeId) <= 0 {
		logx.Error("routeId is nil, chId:", ctx.Channel.GetId())
		return nil, false
	}

	ret, f := b.GetDstChannelMap().Get(routeId)
	if f {
		return ret.(channel.IChannel), f
	} else {
		return nil, false
	}
}

func (b Proxy) QueryAgentChannel(ctx *UpstreamContext) (retChannel channel.IChannel, found bool) {
	return InnerQueryAgentChannel(&b, ctx)
}

func InnerQueryAgentChannel(b IUpstream, ctx *UpstreamContext) (retChannel channel.IChannel, found bool) {
	logx.Info("query agent msg")
	routeId := b.TakeChannnelKey(ctx)
	if len(routeId) <= 0 {
		logx.Error("routeId is nil, chId:", ctx.Channel.GetId())
		return nil, false
	}

	ret, f := b.GetAgentChannelMap().Get(routeId)
	if f {
		return ret.(channel.IChannel), f
	} else {
		return nil, false
	}
}

type UpstreamContext struct {
	AgServer IAgServer
	Channel  channel.IChannel
	Packet   channel.IPacket
	Params   []interface{}
}
