/*
 * 路由是用户自定义方式实现
 * Author:slive
 * DATE:2020/8/13
 */
package agent

import (
	"github.com/Slive/gsfly/channel"
	"github.com/Slive/gsfly/common"
	logx "github.com/Slive/gsfly/logger"
	"github.com/emirpasic/gods/maps/hashmap"
)

// IUpstream upstream接口
type IUpstream interface {
	common.IParent

	GetConf() IUpstreamConf

	GetDstChannelPool() *hashmap.Map

	GetChannelPeerMap() *hashmap.Map

	GetChannelPeer(ctx *UpstreamContext) IChannelPeer

	CreateChannelPeer(ctx *UpstreamContext)

	QueryDstChannel(ctx *UpstreamContext)

	QueryAgentChannel(ctx *UpstreamContext)

	ReleaseByAgentChannel(agentChannel channel.IChannel)

	ReleaseByDstChannel(dstChannel channel.IChannel)

	ReleaseAll()
}

type Upstream struct {
	common.Parent

	// 配置
	Conf IUpstreamConf

	// 记录目标端channel池，可以复用
	DstChannelPool *hashmap.Map

	ChannelPeerMap *hashmap.Map
}

func NewUpstream(parent interface{}, upstreamConf IUpstreamConf) *Upstream {
	if upstreamConf == nil {
		errMsg := "upstreamConf id is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	u := &Upstream{
		Conf: upstreamConf,
		DstChannelPool: hashmap.New(),
		ChannelPeerMap: hashmap.New(),
	}
	u.SetParent(parent)
	return u
}

func (ups *Upstream) GetConf() IUpstreamConf {
	return ups.Conf
}

// GetDstChannelPool 目标channel池, channelId作为主键
func (ups *Upstream) GetDstChannelPool() *hashmap.Map {
	return ups.DstChannelPool
}

// GetChannelPeerMap 获取channel对（agent/dst channel）
func (ups *Upstream) GetChannelPeerMap() *hashmap.Map {
	return ups.ChannelPeerMap
}

// GetChannelPeer 通过UpstreamContext获取到对应的channelpeer
func (ups *Upstream) GetChannelPeer(ctx *UpstreamContext) IChannelPeer {
	panic("implement me")
}

func (proxy *Proxy) QueryDstChannel(ctx *UpstreamContext) {
	InnerQueryDstChannel(proxy, ctx)
}

func (proxy *Proxy) QueryAgentChannel(ctx *UpstreamContext) {
	InnerQueryAgentChannel(proxy, ctx)
}

func InnerQueryDstChannel(b IUpstream, ctx *UpstreamContext) {
	ret := b.GetChannelPeer(ctx)
	if ret != nil {
		peer := ret.(IChannelPeer)
		ctx.SetRet(peer.GetDstChannel())
	} else {
		logx.Warn("query dst is not existed.")
	}
}

func InnerQueryAgentChannel(b IUpstream, ctx *UpstreamContext) {
	ret := b.GetChannelPeer(ctx)
	if ret != nil {
		peer := ret.(IChannelPeer)
		ctx.SetRet(peer.GetAgentChannel())
	} else {
		logx.Warn("query agent is not existed.")
	}
}

func (ups *Upstream) ReleaseAll() {
	upsId := ups.GetConf().GetId()
	defer func() {
		ret := recover()
		if ret != nil {
			logx.Errorf("start to clear upstream, id:%v, ret:%v", upsId, ret)
		} else {
			logx.Info("finish to clear upstream, id:", upsId)
		}
	}()
	logx.Info("start to clear upstream, id:", upsId)
	ups.ChannelPeerMap.Clear()

	// 释放所有dstpoolchannel
	dstChPool := ups.GetDstChannelPool()
	dstVals := dstChPool.Values()
	for _, val := range dstVals {
		val.(channel.IChannel).Stop()
	}
	dstChPool.Clear()
}

type IUpstreamContext interface {
	IProcessContext

	GetRet() channel.IChannel

	SetRet(ret channel.IChannel)
}

type UpstreamContext struct {
	ProcessContext
	ret channel.IChannel
}

func (uCtx *UpstreamContext) GetRet() channel.IChannel {
	return uCtx.ret
}

func (uCtx *UpstreamContext) SetRet(ret channel.IChannel) {
	uCtx.ret = ret
	if ret != nil {
		uCtx.SetOk(true)
	} else {
		uCtx.SetOk(false)
	}
}

func NewUpstreamContext(channel channel.IChannel, packet channel.IPacket, agent bool, params ...interface{}) *UpstreamContext {
	u := &UpstreamContext{}
	u.ProcessContext = *NewProcessContext(channel, packet, agent, params...)
	return u
}

type IChannelPeer interface {
	GetAgentChannel() channel.IChannel

	GetDstChannel() channel.IChannel

	common.IAttact
}

// ChannelPeer channel对（agent/dst channel）
type ChannelPeer struct {
	agentChannel channel.IChannel
	dstChannel   channel.IChannel
	common.Attact
}

func NewChannelPeer(agentChannel, dstChannel channel.IChannel) *ChannelPeer {
	return &ChannelPeer{
		agentChannel: agentChannel,
		dstChannel:   agentChannel,
	}
}

func (cp *ChannelPeer) GetAgentChannel() channel.IChannel {
	return cp.agentChannel
}

func (cp *ChannelPeer) GetDstChannel() channel.IChannel {
	return cp.dstChannel
}
