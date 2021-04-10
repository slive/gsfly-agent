/*
 * 路由是用户自定义方式实现
 * Author:slive
 * DATE:2020/8/13
 */
package agent

import (
	"github.com/slive/gsfly/channel"
	"github.com/slive/gsfly/common"
	logx "github.com/slive/gsfly/logger"
	"github.com/emirpasic/gods/maps/hashmap"
)

// IUpstream upstream接口
type IUpstream interface {
	common.IParent

	GetConf() IUpstreamConf

	// 目标channel列表
	GetDstChannels() *hashmap.Map

	// channel对列表
	GetChannelPeers() *hashmap.Map

	// 获取channel对
	GetChannelPeer(ctx channel.IChHandleContext, isAgent bool) IChannelPeer

	// InitChannelPeer 初始化channel对(agentChannel, dstClientChannel)
	InitChannelPeer(ctx channel.IChHandleContext, params map[string]interface{})

	// QueryDstChannel 查询dstChannel
	QueryDstChannel(ctx channel.IChHandleContext)

	// QueryAgentChannel 查询agentChannel
	QueryAgentChannel(ctx channel.IChHandleContext)

	// ReleaseOnAgentChannel 当agentChannel关闭时，释放相关资源，一般是dstClientChannel
	ReleaseOnAgentChannel(agentCtx channel.IChHandleContext)

	// ReleaseOnDstChannel 当dstClientChannel关闭时，释放相关资源，一般是agentChannel
	ReleaseOnDstChannel(dstCtx channel.IChHandleContext)

	// ReleaseChannelPeers 释放所有channelpeer
	ReleaseChannelPeers()

	// GetExtension 获取扩展点
	GetExtension() IExtension
}

type Upstream struct {
	common.Parent

	// 配置
	conf IUpstreamConf

	// 记录目标端channel，可以复用
	dstChannels *hashmap.Map

	// 记录channelpeer
	channelPeers *hashmap.Map

	// 扩展点
	extension IExtension
}

// NewUpstream 创建upstream对象
// parent 任意父接口，非必选
// upstreamConf upstream配置，必选
// extension 扩展点，见IExtension
func NewUpstream(parent interface{}, upstreamConf IUpstreamConf, extension IExtension) *Upstream {
	if upstreamConf == nil {
		errMsg := "upstreamConf id is nil."
		logx.Error(errMsg)
		panic(errMsg)
	}

	u := &Upstream{
		conf:         upstreamConf,
		dstChannels:  hashmap.New(),
		channelPeers: hashmap.New(),
	}
	u.SetParent(parent)
	if extension == nil {
		// 使用默认扩展点
		u.extension = NewExtension()
	} else {
		u.extension = extension
	}
	return u
}

func (ups *Upstream) GetConf() IUpstreamConf {
	return ups.conf
}

// GetDstChannels 目标channel池, channelId作为主键
func (ups *Upstream) GetDstChannels() *hashmap.Map {
	return ups.dstChannels
}

// GetChannelPeers 获取channel对（agent/dst channel）
func (ups *Upstream) GetChannelPeers() *hashmap.Map {
	return ups.channelPeers
}

// GetChannelPeer 通过UpstreamContext获取到对应的channelpeer
func (ups *Upstream) GetChannelPeer(ctx channel.IChHandleContext, isAgent bool) IChannelPeer {
	panic("implement me")
}

func (ups *Upstream) GetExtension() IExtension {
	return ups.extension
}

func InnerQueryDstChannel(b IUpstream, ctx channel.IChHandleContext) {
	ret := b.GetChannelPeer(ctx, true)
	if ret != nil {
		peer := ret.(IChannelPeer)
		ctx.SetRet(peer.GetDstChannel())
	} else {
		logx.Warn("query dst is not existed.")
	}
}

func InnerQueryAgentChannel(b IUpstream, ctx channel.IChHandleContext) {
	ret := b.GetChannelPeer(ctx, false)
	if ret != nil {
		peer := ret.(IChannelPeer)
		ctx.SetRet(peer.GetAgentChannel())
	} else {
		logx.Warn("query agent is not existed.")
	}
}

func (ups *Upstream) ReleaseChannelPeers() {
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
	ups.channelPeers.Clear()

	// 释放所有dstchannel
	dstChPool := ups.GetDstChannels()
	dstVals := dstChPool.Values()
	for _, val := range dstVals {
		val.(channel.IChannel).Release()
	}
	dstChPool.Clear()
}

// IChannelPeer channel对（agent/dstchannel）
type IChannelPeer interface {
	GetAgentChannel() channel.IChannel

	GetDstChannel() channel.IChannel

	common.IAttact
}

// ChannelPeer channel对（agent/dstchannel)，通过路由后得到的channelpeer
type ChannelPeer struct {
	agentChannel channel.IChannel

	dstChannel channel.IChannel

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
