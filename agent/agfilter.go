/*
 * Author:slive
 * DATE:2020/8/14
 */
package agent

import (
	"github.com/Slive/gsfly/channel"
	"github.com/Slive/gsfly/common"
)

type IFilter interface {
	common.IParent

	common.IId

	GetConf() IFilterConf

	GetPattern() string

	DoFilter()
}

type Filter struct {
	common.Parent

	common.Id

	conf IFilterConf

	pattern string
}

type FilterContext struct {
	Channel    channel.IChannel
	Packet     channel.IPacket
	Params     []interface{}
	RetChannel channel.IChannel
	RetOk      bool
}

func NewFilterContext(channel channel.IChannel, packet channel.IPacket, params ...interface{}) *FilterContext {
	u := &FilterContext{
		Channel: channel,
		Packet:  packet,
		Params:  params,
	}
	return u
}
