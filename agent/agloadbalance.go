/*
 * Author:slive
 * DATE:2020/8/6
 */
package agent

import (
	"errors"
	"github.com/Slive/gsfly/bootstrap"
	"github.com/Slive/gsfly/channel"
	"time"
)

type LoadBalanceType string

const (
	LOADBALANCE_DEFAULT       = LoadBalanceType("default")
	LOADBALANCE_WEIGHT        = LoadBalanceType("weight")
	LOADBALANCE_IPHASH        = LoadBalanceType("iphash")
	LOADBALANCE_IPHASH_WEIGHT = LoadBalanceType("ipsh_weight")
	LOADBALANCE_UNKNOW        = LoadBalanceType("unknown")
)

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
	case LOADBALANCE_DEFAULT.String():
		return LOADBALANCE_DEFAULT, nil
	default:
		return LOADBALANCE_UNKNOW, errors.New("unknown type")
	}
}

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
