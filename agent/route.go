/*
 * 负载均衡路由策略，包括默认策略，权重策略，iphash策略
 * Author:slive
 * DATE:2020/8/2
 */
package agent

import (
	"gsfly/bootstrap"
	"gsfly/channel"
)

type Route interface {
	Select() *bootstrap.BaseClientConf

	Query() *channel.Channel
}

type BaseRoute struct {

}

func (br *BaseRoute) DoRoute() *bootstrap.BaseClientConf {
	return  nil
}

type WeightRoute struct {
	BaseRoute
}

type IpHashRoute struct {
	BaseRoute
}

type DefaultRoute struct {
	BaseRoute
}
