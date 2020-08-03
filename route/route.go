/*
 * 负载均衡路由策略，包括默认策略，权重策略，iphash策略
 * Author:slive
 * DATE:2020/8/2
 */
package route

import "gsfly/bootstrap"

type Route interface {
	DoRoute() bootstrap.ClientConf
}

type BaseRoute struct {
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
