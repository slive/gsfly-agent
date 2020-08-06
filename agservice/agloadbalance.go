/*
 * Author:slive
 * DATE:2020/8/6
 */
package agservice

type LoadBalanceType int

const (
	LOADBALANCE_DEFAULT = iota
	LOADBALANCE_IPHASH
	LOADBALANCE_IPHASH_WEIGHT
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
