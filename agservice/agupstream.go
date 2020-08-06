/*
 * Author:slive
 * DATE:2020/8/6
 */
package agservice

import (
	"gsfly/bootstrap"
	logx "gsfly/logger"
	"time"
)

type AgUpstream interface {
	GetName() string

	GetLoabBalance() AgLoadBalance

	GetClientConfs() []bootstrap.ClientConf

	GetHandleLoadBalance() HandleLoadBalance
}

type BaseAgUpstream struct {
	Name        string
	ClientConfs []bootstrap.ClientConf
	LoadBalance AgLoadBalance
	// 扩展路由
	ExtHandleLoadBalance HandleLoadBalance
	// 内部路由
	HandleLoadBalance HandleLoadBalance
}

func NewBaseAgUpstream(name string, balance AgLoadBalance, handleLoadBalance HandleLoadBalance, confs ...bootstrap.ClientConf) *BaseAgUpstream {
	if len(confs) <= 0 {
		panic("clientconf is nil")
	}

	b := &BaseAgUpstream{
		Name:        name,
		ClientConfs: confs,
	}
	if balance != nil {
		b.LoadBalance = balance
	} else {
		// 取默认路由
		b.LoadBalance = NewBaseAgLoadBalance(LOADBALANCE_DEFAULT)
	}

	// 扩展路由
	if handleLoadBalance != nil {
		b.ExtHandleLoadBalance = handleLoadBalance
	}
	// 内部默认的路由
	b.HandleLoadBalance = InnerHandleLoadBalance

	return b
}

func (b *BaseAgUpstream) GetName() string {
	return b.Name
}

func (b *BaseAgUpstream) GetLoabBalance() AgLoadBalance {
	return b.LoadBalance
}

func (b *BaseAgUpstream) GetClientConfs() []bootstrap.ClientConf {
	return b.ClientConfs
}

func (b *BaseAgUpstream) GetHandleLoadBalance() HandleLoadBalance {
	return b.HandleLoadBalance
}

type HandleLoadBalance func(upstream AgUpstream, params ...interface{}) bootstrap.ClientConf

func InnerHandleLoadBalance(upstream AgUpstream, params ...interface{}) bootstrap.ClientConf {
	var cc bootstrap.ClientConf = nil
	balance := upstream.GetLoabBalance()
	extHandlebalance := upstream.(*BaseAgUpstream).ExtHandleLoadBalance
	if extHandlebalance != nil {
		logx.Debug("start to handle extend loabbalance:", balance)
		cc = extHandlebalance(upstream, params...)
		logx.Debug("finish to handle loabbalance ret:", cc)
	}
	if cc != nil {
		return cc
	}

	logx.Debug("start to handle loabbalance:", balance)
	getType := balance.GetType()
	switch getType {
	case LOADBALANCE_IPHASH:
		break
	case LOADBALANCE_IPHASH_WEIGHT:
	default:
		// LOADBALANCE_DEFAULT
		confs := upstream.GetClientConfs()
		// 求余
		index := time.Now().Second() % len(confs)
		cc = confs[index]
	}
	logx.Debug("handle loabbalance ret:", cc)
	return cc
}
