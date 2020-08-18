/*
 * Author:slive
 * DATE:2020/8/13
 */
package agent

import (
	"gsfly/bootstrap"
	"gsfly/common"
	logx "gsfly/logger"
)

type IServiceConf interface {
	common.IParent

	common.IId

	GetAgServerConf() IAgServerConf

	GetUpstreamConfs() map[string]IUpstreamConf

	GetFilterConfs() map[string]IFilterConf

	GetCommonConf() AgCommonConf

	SetCommonConf(commonConf AgCommonConf)
}

type ServiceConf struct {
	common.Parent

	common.Id

	agServerConf IAgServerConf `json:"conf"`

	upstreamConfs map[string]IUpstreamConf `json:"upstreamConfs"`

	filterConfs map[string]IFilterConf `json:"filterConfs"`

	commonConf AgCommonConf `json:"commonConf"`
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
		agServerConf:  agServerConf,
		upstreamConfs: make(map[string]IUpstreamConf, len(upstreamConfs)),
		filterConfs:   make(map[string]IFilterConf),
	}
	b.SetId(id)
	agServerConf.SetParent(b)

	// 放入缓存
	for _, upConf := range upstreamConfs {
		// 设置父类
		upConf.SetParent(b)
		b.upstreamConfs[upConf.GetId()] = upConf
	}
	logx.Info("finish to NewServiceConf, conf:", b)
	return b
}

func (sc *ServiceConf) GetAgServerConf() IAgServerConf {
	return sc.agServerConf
}

func (sc *ServiceConf) GetUpstreamConfs() map[string]IUpstreamConf {
	return sc.upstreamConfs
}

func (sc *ServiceConf) GetFilterConfs() map[string]IFilterConf {
	return sc.filterConfs
}

func (sc *ServiceConf) GetCommonConf() AgCommonConf {
	return sc.commonConf
}

func (sc *ServiceConf) SetCommonConf(commonConf AgCommonConf) {
	sc.commonConf = commonConf
}

type IAgServerConf interface {
	common.IParent

	common.IId

	GetServerConf() bootstrap.IServerConf

	GetLocationConfs() map[string]ILocationConf
}

type AgServerConf struct {
	common.Id

	common.Parent

	serverConf bootstrap.IServerConf `json:"serverConf"`

	locationConfs map[string]ILocationConf `json:"locationcConfs"`
}

func NewAgServerConf(id string, serverConf bootstrap.IServerConf, locationConf ...ILocationConf) *AgServerConf {
	if len(id) <= 0 {
		errMsg := "service id is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	if serverConf == nil {
		errMsg := "serverConf is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	// TODO 不允许为空
	if locationConf == nil {
		errMsg := "locationConf is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	logx.Info("start to NewAgServerConf, id:", id)
	b := &AgServerConf{
		serverConf:    serverConf,
		locationConfs: make(map[string]ILocationConf, len(locationConf)),
	}
	b.SetId(id)

	// 放入缓存, pattern作为主键
	for _, lConf := range locationConf {
		// 设置父类
		lConf.SetParent(b)
		b.locationConfs[lConf.GetPattern()] = lConf
	}
	logx.Info("finish to NewAgServerConf, conf:", b)
	return b
}

func (asc *AgServerConf) GetServerConf() bootstrap.IServerConf {
	return asc.serverConf
}

func (asc *AgServerConf) GetLocationConfs() map[string]ILocationConf {
	return asc.locationConfs
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

	pattern string `json:"pattern"`

	// 可变配置
	extConf map[string]interface{} `json:"extConf"`
}

func NewFilterConf(id string, pattern string, extConf map[string]interface{}) *FilterConf {
	if len(id) <= 0 {
		errMsg := "service id is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	if &pattern == nil {
		errMsg := "pattern id is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	logx.Info("start to NewFilterConf, id:", id)
	b := &FilterConf{
		pattern: pattern,
		extConf: extConf,
	}
	b.SetId(id)
	logx.Info("finish to NewFilterConf, conf:", b)
	return b
}

func (fc *FilterConf) GetPattern() string {
	return fc.pattern
}

func (fc *FilterConf) GetExtConf() map[string]interface{} {
	return fc.extConf
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

	pattern string `json:"pattern"`

	upstreamId string `json:"upstreamId"`

	// 可变配置
	extConf map[string]interface{} `json:"extConf"`
}

func NewLocationConf(pattern string, upstreamId string, extConf map[string]interface{}) *LocationConf {
	if &pattern == nil {
		errMsg := "pattern is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	if &upstreamId == nil {
		errMsg := "upstreamId is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	logx.Info("start to NewLocationConf, id:", upstreamId)
	b := &LocationConf{
		pattern:    pattern,
		upstreamId: upstreamId,
		extConf:    extConf,
	}
	logx.Info("finish to NewLocationConf, conf:", b)
	return b
}

func (lc *LocationConf) GetUpstreamId() string {
	return lc.upstreamId
}

func (lc *LocationConf) GetPattern() string {
	return lc.pattern
}

func (lc *LocationConf) GetExtConf() map[string]interface{} {
	return lc.extConf
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

	upstreamType UpstreamType `json:"upstreamType"`
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
	GetDstClientConfs() []bootstrap.IClientConf

	GetLoadBalanceType() LoadBalanceType
}

// 常规的（agentChannel）一对(dstChannel)一对等代理方式
type ProxyConf struct {
	UpstreamConf
	// 负载均衡规则
	// LoadBalance ILoadBalance
	// dst客户端配置列表
	DstClientConfs []bootstrap.IClientConf

	// 负载均衡规则
	LoadBalanceType LoadBalanceType
}

func NewProxyConf(id string, loadBalanceType LoadBalanceType, dstClientConfs ...bootstrap.IClientConf) *ProxyConf {
	if dstClientConfs == nil {
		errMsg := "dstClientConfs are nil"
		logx.Error(errMsg)
		panic(errMsg)
	}
	p := &ProxyConf{DstClientConfs: dstClientConfs}
	p.UpstreamConf = *NewUpstreamConf(id, UPSTREAM_PROXY)
	p.LoadBalanceType = loadBalanceType
	return p
}

func (pc *ProxyConf) GetDstClientConfs() []bootstrap.IClientConf {
	return pc.DstClientConfs
}

func (pc *ProxyConf) GetLoadBalanceType() LoadBalanceType {
	return pc.LoadBalanceType
}

// AgCommonConf 代理的通用配置
// 如日志的配置
type AgCommonConf interface {
	// TODO
}