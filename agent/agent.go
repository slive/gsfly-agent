/*
 * Author:slive
 * DATE:2020/7/28
 */
package agent

import (
	logx "github.com/sirupsen/logrus"
	strap "gsfly/bootstrap"
	gch "gsfly/channel"
	glog "gsfly/logger"
	"math/rand"
)

// agent
//  Server
//      network
//      listen
//      config
//
//
//  client
//      network
//      config
//
//      upstream

type Agent interface {
	GetId() string
	GetServerConf() strap.ServerConf
	GetClientConf(clienId string) strap.ClientConf
	GetClientConfs() []strap.ClientConf
	Start() error
	Stop()

	IsClosed() bool

	// GetServer() strap.Server
	// GetDstChannels() map[string]gch.Channel
	// GetDstChannel(srcChId string) gch.Channel
	// AddDstChannel(srcChId string, dstChannel gch.Channel) gch.Channel
	// DelDstChannel(srcChId string)
	//
	// GetSrcChannels() map[string]gch.Channel
	// GetSrcChannel(dstChId string) gch.Channel
	// AddSrcChannel(dstChId string, srcChannel gch.Channel)
	// DelSrcChannel(dstChId string) gch.Channel
}

type BaseAgent struct {
	Server        strap.Server
	ServerConf    strap.ServerConf
	ClientConfMap map[string]strap.ClientConf
	ClientConfs   []strap.ClientConf
	DstChannels   map[string]gch.Channel
	SrcChannels   map[string]gch.Channel
	// 路由规则
	SelectHandle SelectHandle
	QueryHandle  QueryHandle
	Closed       bool
}

func NewBaseAgent(serverConf strap.ServerConf, selectHandle SelectHandle, clientConfs ...strap.ClientConf) *BaseAgent {
	kh := &BaseAgent{
		ServerConf: serverConf,
		Closed:     true,
	}

	clen := len(clientConfs)
	kh.ClientConfs = clientConfs
	kh.ClientConfMap = make(map[string]strap.ClientConf, clen)
	for _, cc := range clientConfs {
		kh.ClientConfMap[cc.GetAddrStr()] = cc
	}

	// 初始化
	kh.DstChannels = make(map[string]gch.Channel, 100)
	kh.SrcChannels = make(map[string]gch.Channel, 100)
	if selectHandle == nil {
		kh.SelectHandle = DefalutSelectHandle
	} else {
		kh.SelectHandle = selectHandle
	}
	return kh
}

func (ag *BaseAgent) GetId() string {
	return "agent:" + ag.ServerConf.GetAddrStr()
}

func (ag *BaseAgent) GetServerConf() strap.ServerConf {
	return ag.ServerConf
}
func (ag *BaseAgent) GetClientConf(clienId string) strap.ClientConf {
	return ag.ClientConfMap[clienId]
}
func (ag *BaseAgent) GetClientConfs() []strap.ClientConf {
	return ag.ClientConfs
}

func (ag *BaseAgent) Start() error {
	panic("unsupport")
}

func (ag *BaseAgent) Stop() {
	id := ag.GetId()
	if ag.IsClosed() {
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
	}()
	logx.Info("start to close agent, id:", id)
	dstChannels := ag.DstChannels
	if dstChannels != nil {
		for key, ch := range dstChannels {
			ch.StopChannel(ch)
			delete(dstChannels, key)
		}
	}

	srcChannels := ag.SrcChannels
	if srcChannels != nil {
		for key, ch := range srcChannels {
			ch.StopChannel(ch)
			delete(srcChannels, key)
		}
	}
	server := ag.Server
	if server != nil {
		server.Stop()
	}
}

func (ag *BaseAgent) IsClosed() bool {
	return ag.Closed
}

func (ka *BaseAgent) SrcCloseFunc(channel gch.Channel) error {
	// 错误处理
	id := channel.GetChId()
	glog.Info("do srcclose, chId:", id)
	dstChannel := ka.DstChannels[id]
	dstChannel.StopChannel(dstChannel)
	if dstChannel != nil {
		dstChannel.StopChannel(dstChannel)
		delete(ka.DstChannels, id)
		delete(ka.SrcChannels, dstChannel.GetChId())
	}
	return nil
}

func (ka *BaseAgent) DstCloseFunc(channel gch.Channel) error {
	// 错误处理
	id := channel.GetChId()
	glog.Info("do dstclose, chId:", id)
	srcChannel := ka.SrcChannels[id]
	if srcChannel != nil {
		srcChannel.StopChannel(srcChannel)
		delete(ka.SrcChannels, id)
		delete(ka.DstChannels, srcChannel.GetChId())
	}

	return nil
}

func DefalutSelectHandle(agent Agent) strap.ClientConf {
	confs := agent.GetClientConfs()
	ccLen := len(confs)
	rand := rand.Int() % ccLen
	conf := confs[rand]
	logx.Infof("rand:%v, conf:%v", rand, conf)
	return conf
}

type SelectHandle func(agent Agent) strap.ClientConf

type QueryHandle func(agent Agent, client strap.ClientConf) strap.Client
