/*
 * Author:slive
 * DATE:2020/8/6
 */
package agservice

import (
	"gsfly/bootstrap"
)

type AgRoute interface {
	GetName() string

	GetLoabBalance() AgLoadBalance

	GetClientConfs() []bootstrap.ClientConf

	GetHandleLoadBalance() LoadBalanceHandle
}

type BaseAgRoute struct {
	Name        string
	ClientConfs []bootstrap.ClientConf
	LoadBalance AgLoadBalance
}

// NewBaseAgUpstream 创建upstream
// extLoadBalanceHandle 扩展的负载均衡实现，可为nil
func NewBaseAgUpstream(name string, balance AgLoadBalance, confs ...bootstrap.ClientConf) *BaseAgRoute {
	if len(confs) <= 0 {
		panic("clientconf is nil")
	}

	b := &BaseAgRoute{
		Name:        name,
		ClientConfs: confs,
	}
	if balance != nil {
		b.LoadBalance = balance
	} else {
		// 取默认路由
		b.LoadBalance = NewBaseAgLoadBalance(LOADBALANCE_DEFAULT)
	}

	return b
}

func (b *BaseAgRoute) GetName() string {
	return b.Name
}

func (b *BaseAgRoute) GetLoabBalance() AgLoadBalance {
	return b.LoadBalance
}

func (b *BaseAgRoute) GetClientConfs() []bootstrap.ClientConf {
	return b.ClientConfs
}

func (b *BaseAgRoute) GetHandleLoadBalance() LoadBalanceHandle {
	return GetLoadBalanceHandle(b.LoadBalance.GetType())
}
