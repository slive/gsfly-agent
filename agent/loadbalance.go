/*
 * Author:slive
 * DATE:2020/8/6
 */
package agent

import (
	"github.com/slive/gsfly/channel"
	"github.com/slive/gsfly/socket"
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

func NewLoadBalance(lbtype LoadBalanceType) ILoadBalance {
	return &LoadBalance{Type: lbtype}
}

func (bl *LoadBalance) GetType() LoadBalanceType {
	return bl.Type
}

// LoadBalanceContext 负载均衡上下文
type LoadBalanceContext struct {
	Agserver     IAgServer
	Upstream     IUpstream
	AgentChannel channel.IChannel
	Result       socket.IClientConf
}

func NewLoadBalanceContext(agserver IAgServer, agRoute IUpstream, agentChannel channel.IChannel) *LoadBalanceContext {
	return &LoadBalanceContext{
		Agserver:     agserver,
		Upstream:     agRoute,
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

// String 获取协议对应的字符串
func (p LoadBalanceType) String() string {
	switch p {
	case LOADBALANCE_DEFAULT:
		return "default"
	case LOADBALANCE_WEIGHT:
		return "weight"
	case LOADBALANCE_IPHASH:
		return "iphash"
	case LOADBALANCE_IPHASH_WEIGHT:
		return "ipsh_weight"
	default:
		return "unknown"
	}
}

func GetLoadBalanceType(lbtype string) (LoadBalanceType, error) {
	switch lbtype {
	case LOADBALANCE_IPHASH.String():
		return LOADBALANCE_IPHASH, nil
	case LOADBALANCE_WEIGHT.String():
		return LOADBALANCE_WEIGHT, nil
	case LOADBALANCE_IPHASH_WEIGHT.String():
		return LOADBALANCE_IPHASH_WEIGHT, nil
	default:
		return LOADBALANCE_DEFAULT, nil

	}
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
	confs := upstream.GetConf().(IProxyConf).GetDstClientConfs()
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
