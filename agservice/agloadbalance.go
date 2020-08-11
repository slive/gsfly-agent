/*
 * Author:slive
 * DATE:2020/8/6
 */
package agservice

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

type AgLoadBalance interface {
	GetType() LoadBalanceType
}

type BaseAgLoadBalance struct {
	Type LoadBalanceType
}

func NewBaseAgLoadBalance(lbtype LoadBalanceType) AgLoadBalance {
	return &BaseAgLoadBalance{Type: lbtype}
}

func (bl *BaseAgLoadBalance) GetType() LoadBalanceType {
	return bl.Type
}

// LoadBalanceContext 负载均衡上下文
type LoadBalanceContext struct {
	Agserver     AgServer
	Upstream     AgRoute
	AgentChannel channel.Channel
	Result       bootstrap.ClientConf
}

func NewLoadBalanceContext(agserver AgServer, upstream AgRoute, agentChannel channel.Channel) *LoadBalanceContext {
	return &LoadBalanceContext{
		Agserver:     agserver,
		Upstream:     upstream,
		AgentChannel: agentChannel,
		Result:       nil,
	}
}

// 负载均衡处理handle
type LoadBalanceHandle func(lbcontext *LoadBalanceContext)

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

func defaultLoadBalanceHandle(bcontext *LoadBalanceContext) {
	upstream := bcontext.Upstream
	confs := upstream.GetClientConfs()
	// 求余
	index := time.Now().Second() % len(confs)
	bcontext.Result = confs[index]
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
