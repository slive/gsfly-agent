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

	// 目标channel列表
	GetDstChannelMap() *hashmap.Map

	// channel对列表
	GetChannelPeerMap() *hashmap.Map

	// 获取channel对
	GetChannelPeer(ctx *UpstreamContext) IChannelPeer

	// 初始化channel对
	InitChannelPeer(ctx *UpstreamContext)

	QueryDstChannel(ctx *UpstreamContext)

	QueryAgentChannel(ctx *UpstreamContext)

	ReleaseByAgentChannel(agentChannel channel.IChannel)

	ReleaseByDstChannel(dstChannel channel.IChannel)

	ReleaseAll()

	SetTransfer(transfer IExtension)
	GetTransfer() IExtension
}

type Upstream struct {
	common.Parent

	// 配置
	Conf IUpstreamConf

	// 记录目标端channel池，可以复用
	DstChannelMap *hashmap.Map

	ChannelPeerMap *hashmap.Map

	transfer IExtension
}

func NewUpstream(parent interface{}, upstreamConf IUpstreamConf, transfer IExtension) *Upstream {
	if upstreamConf == nil {
		errMsg := "upstreamConf id is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	u := &Upstream{
		Conf:           upstreamConf,
		DstChannelMap:  hashmap.New(),
		ChannelPeerMap: hashmap.New(),
	}
	u.SetParent(parent)
	if transfer == nil {
		u.transfer = NewExtension()
	} else {
		u.transfer = transfer
	}
	return u
}

func (ups *Upstream) GetConf() IUpstreamConf {
	return ups.Conf
}

// GetDstChannelMap 目标channel池, channelId作为主键
func (ups *Upstream) GetDstChannelMap() *hashmap.Map {
	return ups.DstChannelMap
}

// GetChannelPeerMap 获取channel对（agent/dst channel）
func (ups *Upstream) GetChannelPeerMap() *hashmap.Map {
	return ups.ChannelPeerMap
}

// GetChannelPeer 通过UpstreamContext获取到对应的channelpeer
func (ups *Upstream) GetChannelPeer(ctx *UpstreamContext) IChannelPeer {
	panic("implement me")
}

func (ups *Upstream) SetTransfer(transfer IExtension) {
	ups.transfer = transfer
}
func (ups *Upstream) GetTransfer() IExtension {
	return ups.transfer
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
	dstChPool := ups.GetDstChannelMap()
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
		dstChannel:   dstChannel,
	}
}

func (cp *ChannelPeer) GetAgentChannel() channel.IChannel {
	return cp.agentChannel
}

func (cp *ChannelPeer) GetDstChannel() channel.IChannel {
	return cp.dstChannel
}
