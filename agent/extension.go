/*
 * Author:slive
 * DATE:2020/10/14
 */
package agent

import (
	gch "github.com/Slive/gsfly/channel"
	"github.com/Slive/gsfly/channel/tcpx"
	"github.com/Slive/gsfly/common"
	logx "github.com/Slive/gsfly/logger"
	"github.com/Slive/gsfly/socket"
	"github.com/gorilla/websocket"
)

type IExtension interface {
	common.IParent
	Transfer(fromCtx gch.IChHandlerContext, toChannel gch.IChannel)

	GetLocationPattern(ctx gch.IChHandlerContext) (localPattern string, params []interface{})

	InitUpstream(upsConf IUpstreamConf) IUpstream
}

type Extension struct {
	common.Parent
}

func NewExtension() *Extension {
	e := &Extension{}
	e.Parent = *common.NewParent(nil)
	return e
}

func (t *Extension) Transfer(fromCtx gch.IChHandlerContext, toChannel gch.IChannel) {
	fromPacket := fromCtx.GetPacket()
	fromChannel := fromPacket.GetChannel()
	dstPacket := toChannel.NewPacket()
	fromNetwork := fromChannel.GetConf().GetNetwork()
	toNetwork := toChannel.GetConf().GetNetwork()
	// 根据不同的协议类型，转发到不同的目的toChannel
	switch fromNetwork {
	case gch.NETWORK_WS:
		switch toNetwork {
		case gch.NETWORK_WS:
			dstPacket.SetData(fromPacket.GetData())
			// 同样的协议保持包类型一致
			dstPacket.(*tcpx.WsPacket).MsgType = fromPacket.(*tcpx.WsPacket).MsgType
		default:
			dstPacket.SetData(fromPacket.GetData())
		}
	case gch.NETWORK_KCP:
		switch toNetwork {
		case gch.NETWORK_WS:
			dstPacket.SetData(fromPacket.GetData())
			// 默认text方式？，pingpong等方式呢？
			dstPacket.(*tcpx.WsPacket).MsgType = websocket.TextMessage
		default:
			dstPacket.SetData(fromPacket.GetData())
		}
	default:
		dstPacket.SetData(fromPacket.GetData())
	}
	toChannel.Write(dstPacket)
}

func (t *Extension) GetLocationPattern(ctx gch.IChHandlerContext) (localPattern string, params []interface{}) {
	agentChannel := ctx.GetChannel()
	protocol := agentChannel.GetConf().GetNetwork()
	localPattern = ""
	params = make([]interface{}, 1)
	switch protocol {
	case gch.NETWORK_WS:
		wsChannel := agentChannel.(*tcpx.WsChannel)
		params[0] = wsChannel.GetParams()
		// 1、约定用path来限定路径
		wsServerConf := wsChannel.GetConf().(socket.IWsServerConf)
		localPattern = wsServerConf.GetPath()
	default:
		// TODO 待完善
		localPattern = default_localPattern
	}
	logx.Infof("channel params:%v, localPattern:%v", params, localPattern)
	return localPattern, params
}

func (t *Extension) InitUpstream(upsConf IUpstreamConf) IUpstream {
	// 不同的upstreamtype进行不同的处理
	upsType := upsConf.GetUpstreamType()
	var ups IUpstream
	if upsType == UPSTREAM_PROXY {
		proxyConf, ok := upsConf.(IProxyConf)
		if ok {
			ups = NewProxy(t.GetParent(), proxyConf, t)
		} else {
			panic("upstream type is invalid")
		}
	} else {
		// TODO
	}
	return ups
}