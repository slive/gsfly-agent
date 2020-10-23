/*
 * 代理服务类
 * Author:slive
 * DATE:2020/8/6
 */
package agent

import (
	"errors"
	"github.com/Slive/gsfly/channel"
	logx "github.com/Slive/gsfly/logger"
)

// IService 代理服务
type IService interface {
	GetAgServer() IAgServer

	GetConf() IServiceConf

	GetUpstreams() map[string]IUpstream

	// GetFilters() map[string]IFilter

	Start() error

	Stop()

	IsClosed() bool

	// InitUpstream(upsConf IUpstreamConf) IUpstream

	// CreateFilter(filterConf IFilterConf) IFilter
}

type Service struct {
	AgServer IAgServer

	ServiceConf IServiceConf

	Closed bool

	Upstreams map[string]IUpstream

	extension IExtension

	// Filters map[string]IFilter
}

func NewService(serviceConf IServiceConf, extension IExtension) *Service {
	if serviceConf == nil {
		errMsg := "serviceConf is nil"
		logx.Error(errMsg)
		panic(errMsg)
	}

	service := &Service{ServiceConf: serviceConf}
	if extension != nil {
		service.extension = extension
	} else {
		service.extension = NewExtension()
	}
	service.extension.SetParent(service)

	// 初始化upstreams
	upsConfs := serviceConf.GetUpstreamConfs()
	service.Upstreams = make(map[string]IUpstream, len(upsConfs))
	for key, conf := range upsConfs {
		ups := service.extension.InitUpstream(conf, service.extension)
		if ups != nil {
			service.GetUpstreams()[key] = ups
		} else {
			logx.Warnf("create ups is nil, conf:", conf)
		}
	}

	// 初始化filters
	// filterConfs := serviceConf.GetFilterConfs()
	// service.Filters = make(map[string]IFilter, len(filterConfs))
	// for key, conf := range filterConfs {
	// 	filter := service.CreateFilter(conf)
	// 	if filter != nil {
	// 		service.GetFilters()[key] = filter
	// 	}
	// }
	service.Closed = true
	return service
}

func (service *Service) GetAgServer() IAgServer {
	return service.AgServer
}

func (service *Service) GetUpstreams() map[string]IUpstream {
	return service.Upstreams
}

// func (service *Service) GetFilters() map[string]IFilter {
// 	return service.Filters
// }

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
	agServer := NewAgServer(service, agServerConf, service.extension)
	err := agServer.Listen()
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
		server.Close()
	}

	// 清理upstream相关
	upstreams := service.Upstreams
	for _, upstream := range upstreams {
		upstream.ReleaseChannelPeers()
		service.Upstreams = nil
	}

	// 清理filter相关
	// service.Filters = nil
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
	IsAgent() bool
}

type Input struct {
	Channel channel.IChannel
	Packet  channel.IPacket
	Agent   bool
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

func (input *Input) IsAgent() bool {
	return input.Agent
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

func NewProcessContext(channel channel.IChannel, packet channel.IPacket, agent bool, params ...interface{}) *ProcessContext {
	u := &ProcessContext{}
	u.Input = Input{
		Channel: channel,
		Packet:  packet,
		Agent:   agent,
		Params:  params,
	}
	return u
}
