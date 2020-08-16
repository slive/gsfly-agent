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
	Agserver     IAgServer
	Proxy        IProxy
	AgentChannel channel.IChannel
}

func NewLoadBalanceContext(agserver IAgServer, proxy IProxy, agentChannel channel.IChannel) *LoadBalanceContext {
	return &LoadBalanceContext{
		Agserver:     agserver,
		Proxy:        proxy,
		AgentChannel: agentChannel,
	}
}

// 负载均衡处理handle
type LoadBalanceHandle func(lbCtx *LoadBalanceContext) bootstrap.IClientConf

func init() {
	AddLoadBalanceHandle(LOADBALANCE_DEFAULT, defaultLoadBalanceHandle)
	AddLoadBalanceHandle(LOADBALANCE_WEIGHT, weighLoadBalanceHandle)
	AddLoadBalanceHandle(LOADBALANCE_IPHASH, iphashLoadBalanceHandle)
	AddLoadBalanceHandle(LOADBALANCE_IPHASH_WEIGHT, iphashweighLoadBalanceHandle)
}

var localBalanceHandles = make(map[LoadBalanceType]LoadBalanceHandle)

func GetLoadBalanceHandle(lbtype LoadBalanceType) LoadBalanceHandle {
	return localBalanceHandles[lbtype]
}

func AddLoadBalanceHandle(lbtype LoadBalanceType, loadBalanceHandle LoadBalanceHandle) {
	localBalanceHandles[lbtype] = loadBalanceHandle
}

func defaultLoadBalanceHandle(bcontext *LoadBalanceContext) bootstrap.IClientConf {
	upstream := bcontext.Proxy
	confs := upstream.GetConf().(IProxyConf).GetDstClientConfs()
	// 求余
	index := time.Now().Second() % len(confs)
	return confs[index]
}

func weighLoadBalanceHandle(bcontext *LoadBalanceContext) bootstrap.IClientConf {
	// TODO 按照比例来
	return nil
}

func iphashLoadBalanceHandle(bcontext *LoadBalanceContext) bootstrap.IClientConf {
	// TODO iphash
	return nil
}

func iphashweighLoadBalanceHandle(bcontext *LoadBalanceContext) bootstrap.IClientConf {
	// TODO iphash比重
	return nil
}
