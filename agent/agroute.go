/*
 * Author:slive
 * DATE:2020/9/16
 */
package agent

import (
	"github.com/Slive/gsfly/channel"
	logx "github.com/Slive/gsfly/logger"
	"github.com/Slive/gsfly/util"
)

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
		poolIndex := util.Hashcode(routeId) % maxDstChannelSize
		values := route.DstChannelPool.Values()
		if poolIndex < len(values) {
			retChannel = values[poolIndex].(channel.IChannel)
		}
	}

	hadcreated := (retChannel != nil)
	if hadcreated {
		// route.GetDstChannelMap().Put(routeId, retChannel)
		// route.GetAgentChannelMap().Put(routeId, ctx.Channel)
		ctx.SetRet(retChannel)
	} else {
		// TODO 创建...
		panic("wait create dstchannel")
		// route.DstChannelMap.Put(routeId, retChannel)
	}
}
