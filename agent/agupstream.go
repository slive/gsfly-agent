/*
 * 路由是用户自定义方式实现
 * Author:slive
 * DATE:2020/8/13
 */
package agent

import (
	"github.com/emirpasic/gods/maps/hashmap"
	"gsfly/channel"
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

	SelectDstChannel(ctx *UpstreamContext)

	QueryDstChannel(ctx *UpstreamContext)

	QueryAgentChannel(ctx *UpstreamContext)

	ClearAgentChannel(agentChannel channel.IChannel)

	ClearDstChannel(dstChannel channel.IChannel)

	Clear()
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

func (ups *Upstream) GetConf() IUpstreamConf {
	return ups.Conf
}

// GetDstChannelPool 目标channel池, channelId作为主键
func (ups *Upstream) GetDstChannelPool() *hashmap.Map {
	return ups.DstChannelPool
}

// GetDstChannelMap 记录AgentChannel, 主键为路由的出来的Id
func (ups *Upstream) GetDstChannelMap() *hashmap.Map {
	return ups.DstChannelMap
}

// GetAgentChannelMap 记录AgentChannel, 主键为路由的出来的Id
func (ups *Upstream) GetAgentChannelMap() *hashmap.Map {
	return ups.AgentChannelMap
}

func (ups *Upstream) TakeChannnelKey(ctx *UpstreamContext) (routeId string) {
	panic("implement me")
}

func (ups *Upstream) ClearAgentChannel(dstChannel channel.IChannel) {
	panic("implement me")
}

func (ups *Upstream) ClearDstChannel(dstChannel channel.IChannel) {
	panic("implement me")
}

type Route struct {
	Upstream
}

func (route *Route) TakeChannnelKey(ctx *UpstreamContext) (routeId string) {
	panic("implement me")

}

func (route *Route) SelectDstChannel(ctx *UpstreamContext) {
	routeId := route.TakeChannnelKey(ctx)
	if len(routeId) <= 0 {
		logx.Error("routeId is nil, chId:", ctx.GetChannel().GetId())
		return
	}

	var retChannel channel.IChannel
	maxDstChannelSize := 0
	if maxDstChannelSize > 0 {
		// 有dstchannel限制
		pool := util.Hashcode(routeId) % maxDstChannelSize
		values := route.DstChannelPool.Values()
		if pool < len(values) {
			retChannel = values[pool].(channel.IChannel)
		}
	}

	hadcreated := (retChannel != nil)
	if hadcreated {
		route.GetDstChannelMap().Put(routeId, retChannel)
		route.GetAgentChannelMap().Put(routeId, ctx.Channel)
		ctx.SetRet(retChannel)
	} else {
		// TODO 创建...
		panic("wait create dstchannel")
		// route.DstChannelMap.Put(routeId, retChannel)
	}
}

func InnerQueryDstChannel(b IUpstream, ctx *UpstreamContext) {
	logx.Info("query dst msg")
	routeId := b.TakeChannnelKey(ctx)
	if len(routeId) <= 0 {
		logx.Error("routeId is nil, chId:", ctx.Channel.GetId())
		return
	}

	ret, f := b.GetDstChannelMap().Get(routeId)
	if f {
		ctx.SetRet(ret.(channel.IChannel))
	} else {
		return
	}
}

func InnerQueryAgentChannel(b IUpstream, ctx *UpstreamContext) {
	logx.Info("query agent msg")
	routeId := b.TakeChannnelKey(ctx)
	if len(routeId) <= 0 {
		logx.Error("routeId is nil, chId:", ctx.Channel.GetId())
		return
	}

	ret, f := b.GetAgentChannelMap().Get(routeId)
	if f {
		ctx.SetRet(ret.(channel.IChannel))
	}
}

func (ups *Upstream) Clear() {
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
	ups.GetDstChannelMap().Clear()
	ups.GetAgentChannelMap().Clear()
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

func NewUpstreamContext(channel channel.IChannel, packet channel.IPacket, params ...interface{}) *UpstreamContext {
	u := &UpstreamContext{}
	u.ProcessContext = *NewProcessContext(channel, packet, params...)
	return u
}
