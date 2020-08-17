/*
 * Author:slive
 * DATE:2020/8/6
 */
package agent

import (
	"gsfly/bootstrap"
	"gsfly/channel"
	"time"
)

type LoadBalanceType int

const (
	LOADBALANCE_DEFAULT       = LoadBalanceType(0)
	LOADBALANCE_WEIGHT        = LoadBalanceType(1)
	LOADBALANCE_IPHASH        = LoadBalanceType(2)
	LOADBALANCE_IPHASH_WEIGHT = LoadBalanceType(3)
)

// ILoadBalance 负载均衡接口
type ILoadBalance interface {
	GetType() LoadBalanceType
}

type LoadBalance struct {
	Type LoadBalanceType
}

func NewLoadBalance(lbtype LoadBalanceType) *LoadBalance {
	return &LoadBalance{Type: lbtype}
}

func (bl *LoadBalance) GetType() LoadBalanceType {
	return bl.Type
}

// LoadBalanceContext 负载均衡上下文
type LoadBalanceContext struct {
	Agserver      IAgServer
	Proxy         IProxy
	AgentChannel  channel.IChannel
	DstClientConf bootstrap.IClientConf
}

func NewLbContext(agserver IAgServer, proxy IProxy, agentChannel channel.IChannel) *LoadBalanceContext {
	return &LoadBalanceContext{
		Agserver:     agserver,
		Proxy:        proxy,
		AgentChannel: agentChannel,
	}
}

// 负载均衡处理handle
type LoadBalanceHandle func(lbCtx *LoadBalanceContext)

func init() {
	AddLbHandle(LOADBALANCE_DEFAULT, defaultLoadBalanceHandle)
	AddLbHandle(LOADBALANCE_WEIGHT, weighLoadBalanceHandle)
	AddLbHandle(LOADBALANCE_IPHASH, iphashLoadBalanceHandle)
	AddLbHandle(LOADBALANCE_IPHASH_WEIGHT, iphashweighLoadBalanceHandle)
}

var localBalanceHandles = make(map[LoadBalanceType]LoadBalanceHandle)

func GetLbHandle(lbtype LoadBalanceType) LoadBalanceHandle {
	return localBalanceHandles[lbtype]
}

func AddLbHandle(lbtype LoadBalanceType, loadBalanceHandle LoadBalanceHandle) {
	localBalanceHandles[lbtype] = loadBalanceHandle
}

func defaultLoadBalanceHandle(lbCtx *LoadBalanceContext) {
	upstream := lbCtx.Proxy
	confs := upstream.GetConf().(IProxyConf).GetDstClientConfs()
	// 求余
	index := time.Now().Second() % len(confs)
	lbCtx.DstClientConf = confs[index]
}

func weighLoadBalanceHandle(bcontext *LoadBalanceContext) {
	// TODO 按照比例来
}

func iphashLoadBalanceHandle(bcontext *LoadBalanceContext) {
	// TODO iphash
}

func iphashweighLoadBalanceHandle(bcontext *LoadBalanceContext) {
	// TODO iphash比重
}
