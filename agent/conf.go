/*
 * Author:slive
 * DATE:2020/8/13
 */
package agent

import (
	"github.com/Slive/gsfly/common"
	logx "github.com/Slive/gsfly/logger"
	"github.com/Slive/gsfly/socket"
	"sync"
)

type IServiceConf interface {
	common.IParent

	common.IId

	GetAgServerConf() IAgServerConf

	GetUpstreamConfs() map[string]IUpstreamConf

	GetFilterConfs() map[string]IFilterConf
}

type ServiceConf struct {
	common.Parent

	common.Id

	AgServerConf IAgServerConf

	UpstreamConfs map[string]IUpstreamConf

	FilterConfs map[string]IFilterConf
}

func NewServiceConf(id string, agServerConf IAgServerConf, upstreamConfs ...IUpstreamConf) *ServiceConf {
	if len(id) <= 0 {
		errMsg := "service id is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	if agServerConf == nil {
		errMsg := "conf is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	if upstreamConfs == nil {
		errMsg := "upstream is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	logx.Info("start to NewServiceConf, id:", id)

	b := &ServiceConf{
		AgServerConf:  agServerConf,
		UpstreamConfs: make(map[string]IUpstreamConf, len(upstreamConfs)),
		FilterConfs:   make(map[string]IFilterConf),
	}
	b.SetId(id)
	agServerConf.SetParent(b)

	// 放入缓存
	for _, upConf := range upstreamConfs {
		// 设置父类
		upConf.SetParent(b)
		b.UpstreamConfs[upConf.GetId()] = upConf
	}
	logx.Info("finish to NewServiceConf, conf:", b)
	return b
}

func (sc *ServiceConf) GetAgServerConf() IAgServerConf {
	return sc.AgServerConf
}

func (sc *ServiceConf) GetUpstreamConfs() map[string]IUpstreamConf {
	return sc.UpstreamConfs
}

func (sc *ServiceConf) GetFilterConfs() map[string]IFilterConf {
	return sc.FilterConfs
}

type IAgServerConf interface {
	socket.IServerConf

	GetServerConf() socket.IServerConf

	GetLocationConfs() map[string]ILocationConf
}

type AgServerConf struct {

	socket.IServerConf

	LocationConfs []ILocationConf

	locationConfMap map[string]ILocationConf

	locationOne sync.Once
}

func NewAgServerConf(id string, serverConf socket.IServerConf, locationConfs ...ILocationConf) *AgServerConf {
	if serverConf == nil {
		errMsg := "ServerConf is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	// TODO 不允许为空
	if locationConfs == nil {
		errMsg := "locationConfs are nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	logx.Info("start to NewAgServerConf, id:", id)
	b := &AgServerConf{
		IServerConf:  serverConf,
		LocationConfs: locationConfs,
	}

	b.SetId(id)
	logx.Info("finish to NewAgServerConf, conf:", b)
	return b
}

func (asc *AgServerConf) initLocationConfMap() {
	// 放入缓存, pattern作为主键
	locationConfs := asc.LocationConfs
	asc.locationConfMap = make(map[string]ILocationConf, len(locationConfs))
	for _, lConf := range locationConfs {
		// 设置父类
		lConf.SetParent(asc)
		asc.locationConfMap[lConf.GetPattern()] = lConf
	}
}

func (asc *AgServerConf) GetServerConf() socket.IServerConf {
	return asc.IServerConf
}

func (asc *AgServerConf) GetLocationConfs() map[string]ILocationConf {
	asc.locationOne.Do(asc.initLocationConfMap)
	return asc.locationConfMap
}

// IFilterConf 过滤器的配置，根据pattern找到对应的filter，然后获取到filter进行处理
type IFilterConf interface {
	common.IParent

	common.IId

	GetPattern() string

	GetExtConf() map[string]interface{}
}

// 可实现访问认证，如会话，权限，ip黑白名单规则等
type FilterConf struct {
	common.Parent

	common.Id

	Pattern string

	// 可变配置
	ExtConf map[string]interface{}
}

func NewFilterConf(id string, pattern string, extConf map[string]interface{}) *FilterConf {
	if len(id) <= 0 {
		errMsg := "service id is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	if &pattern == nil {
		errMsg := "Pattern id is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	logx.Info("start to NewFilterConf, id:", id)
	b := &FilterConf{
		Pattern: pattern,
		ExtConf: extConf,
	}
	b.SetId(id)
	logx.Info("finish to NewFilterConf, conf:", b)
	return b
}

func (fc *FilterConf) GetPattern() string {
	return fc.Pattern
}

func (fc *FilterConf) GetExtConf() map[string]interface{} {
	return fc.ExtConf
}

// ILocationConf 定位的配置，根据pattern找到对应的upstreamId，然后获取到upstream进行处理
type ILocationConf interface {
	common.IParent

	GetPattern() string

	GetUpstreamId() string

	GetExtConf() map[string]interface{}
}

type LocationConf struct {
	common.Parent

	Pattern string

	UpstreamId string

	// 可变配置
	ExtConf map[string]interface{}
}

func NewLocationConf(pattern string, upstreamId string, extConf map[string]interface{}) *LocationConf {
	if &pattern == nil {
		errMsg := "Pattern is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	if &upstreamId == nil {
		errMsg := "UpstreamId is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	logx.Info("start to NewLocationConf, id:", upstreamId)
	b := &LocationConf{
		Pattern:    pattern,
		UpstreamId: upstreamId,
		ExtConf:    extConf,
	}
	logx.Info("finish to NewLocationConf, conf:", b)
	return b
}

func (lc *LocationConf) GetUpstreamId() string {
	return lc.UpstreamId
}

func (lc *LocationConf) GetPattern() string {
	return lc.Pattern
}

func (lc *LocationConf) GetExtConf() map[string]interface{} {
	return lc.ExtConf
}

type UpstreamType string

const (
	UPSTREAM_PROXY = "proxy"
	UPSTREAM_ROUTE = "route"
)

// IUpstreamConf upstream包括如下几种场景：
// 1、可实现代理功能
// 2、可实现路由功能
type IUpstreamConf interface {
	common.IParent

	common.IId

	GetUpstreamType() UpstreamType

	SetUpstreamType(upstreamType UpstreamType)
}

type UpstreamConf struct {
	common.Id

	common.Parent

	upstreamType UpstreamType `json:"type"`
}

func NewUpstreamConf(id string, upstreamType UpstreamType) *UpstreamConf {
	if len(id) <= 0 {
		errMsg := "service id is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	if &upstreamType == nil {
		upstreamType = UPSTREAM_PROXY
	}

	logx.Info("start to NewUpstreamConf, id:", id)
	b := &UpstreamConf{
		upstreamType: upstreamType,
	}
	b.Id = *common.NewId()
	b.SetId(id)
	logx.Info("finish to NewUpstreamConf, conf:", b)
	return b
}

func (upc *UpstreamConf) GetUpstreamType() UpstreamType {
	return upc.upstreamType
}

func (upc *UpstreamConf) SetUpstreamType(upstreamType UpstreamType) {
	upc.upstreamType = upstreamType
}

type IProxyConf interface {
	IUpstreamConf
	GetDstClientConfs() []socket.IClientConf

	GetLoadBalanceType() LoadBalanceType
}

// 常规的（agentChannel）一对(dstChannel)一对等代理方式
type ProxyConf struct {
	UpstreamConf
	// 负载均衡规则
	// LoadBalance ILoadBalance
	// dst客户端配置列表
	DstClientConfs []socket.IClientConf

	// 负载均衡规则
	LoadBalanceType LoadBalanceType
}

func NewProxyConf(id string, loadBalanceType LoadBalanceType, dstClientConfs ...socket.IClientConf) *ProxyConf {
	if dstClientConfs == nil {
		errMsg := "dstClientConfs are nil"
		logx.Error(errMsg)
		panic(errMsg)
	}
	p := &ProxyConf{DstClientConfs: make([]socket.IClientConf, len(dstClientConfs))}
	for index, conf := range dstClientConfs {
		p.DstClientConfs[index] = conf
	}
	p.UpstreamConf = *NewUpstreamConf(id, UPSTREAM_PROXY)
	p.LoadBalanceType = loadBalanceType
	return p
}

func (pc *ProxyConf) GetDstClientConfs() []socket.IClientConf {
	return pc.DstClientConfs
}

func (pc *ProxyConf) GetLoadBalanceType() LoadBalanceType {
	return pc.LoadBalanceType
}
