/*
 * 代理服务类
 * Author:slive
 * DATE:2020/8/6
 */
package agent

import (
	"errors"
	"gsfly/channel"
	logx "gsfly/logger"
)

// IService 代理服务
type IService interface {
	GetAgServer() IAgServer

	GetConf() IServiceConf

	GetUpstreams() map[string]IUpstream

	GetFilters() map[string]IFilter

	Start() error

	Stop()

	IsClosed() bool
}

type Service struct {
	AgServer IAgServer

	ServiceConf IServiceConf

	Closed bool

	Upstreams map[string]IUpstream

	Filters map[string]IFilter
}

func NewService(serviceConf IServiceConf) *Service {
	if serviceConf == nil {
		errMsg := "serviceConf is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	b := &Service{ServiceConf: serviceConf}
	upsConfs := serviceConf.GetUpstreamConfs()
	b.Upstreams = make(map[string]IUpstream, len(upsConfs))
	for key, conf := range upsConfs {
		// 不同的upstreamtype进行不同的处理
		upsType := conf.GetUpstreamType()
		if upsType == UPSTREAM_PROXY {
			proxyConf, ok := conf.(IProxyConf)
			if ok {
				b.Upstreams[key] = NewProxy(b, proxyConf)
			} else {
				panic("upstream type is invalid")
			}
		} else {
			// TODO
		}
	}
	b.Closed = true
	return b
}

func (service *Service) GetAgServer() IAgServer {
	return service.AgServer
}

func (service *Service) GetUpstreams() map[string]IUpstream {
	return service.Upstreams
}

func (service *Service) GetFilters() map[string]IFilter {
	return service.Filters
}

func (service *Service) Start() error {
	id := service.GetConf().GetId()
	if !service.IsClosed() {
		return errors.New("agentservice had start, id:" + id)
	}

	defer func() {
		ret := recover()
		if ret != nil {
			logx.Errorf("finish to start agent, id:%v, msg:%v", id, ret)
		} else {
			logx.Info("finish to start agent, id:", id)
		}
	}()
	logx.Info("start to agent service, id:", id)
	agServerConf := service.GetConf().GetAgServerConf()
	agServer := NewAgServer(service, agServerConf)
	err := agServer.Start()
	if err == nil {
		service.AgServer = agServer
		service.Closed = true
	}
	return err
}

func (service *Service) Stop() {
	id := service.GetConf().GetId()
	if service.IsClosed() {
		logx.Info("had close agent, id:", id)
		return
	}

	defer func() {
		ret := recover()
		if ret != nil {
			logx.Errorf("finish to close agent, id:%v, msg:%v", id, ret)
		} else {
			logx.Info("finish to close agent, id:", id)
		}
		service.Closed = true
	}()
	logx.Info("start to close agent, id:", id)
	// 清理代理服务
	server := service.GetAgServer()
	if server != nil {
		server.Stop()
	}

	// 清理upstream相关
	upstreams := service.Upstreams
	for _, upstream := range upstreams {
		upstream.Clear()
	}
}

func (b *Service) IsClosed() bool {
	return b.Closed
}

func (b *Service) GetConf() IServiceConf {
	return b.ServiceConf
}

type IInput interface {
	GetChannel() channel.IChannel
	GetPacket() channel.IPacket
	GetParams() []interface{}
}

type Input struct {
	Channel channel.IChannel
	Packet  channel.IPacket
	Params  []interface{}
}

func (input *Input) GetChannel() channel.IChannel {
	return input.Channel
}

func (input *Input) GetPacket() channel.IPacket {
	return input.Packet
}

func (input *Input) GetParams() []interface{} {
	return input.Params
}

type IOutput interface {
	IsOk() bool
	SetOk(ok bool)
}

type Output struct {
	ok bool
}

func (output *Output) IsOk() bool {
	return output.ok
}

func (output *Output) SetOk(ok bool) {
	output.ok = ok
}

type IProcessContext interface {
	IInput
	IOutput
}

type ProcessContext struct {
	Input
	Output
}

func NewProcessContext(channel channel.IChannel, packet channel.IPacket, params ...interface{}) *ProcessContext {
	u := &ProcessContext{}
	u.Input = Input{
		Channel: channel,
		Packet:  packet,
		Params:  params,
	}
	return u
}
