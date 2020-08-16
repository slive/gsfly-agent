/*
 * Author:slive
 * DATE:2020/8/6
 */
package agent

import (
	"gsfly/bootstrap"
	"gsfly/common"
	logx "gsfly/logger"
)

type IAgServer interface {
	common.IParent

	GetServer() bootstrap.IServerStrap

	GetAgServerConf() IAgServerConf

	GetLocationHandle() LocationHandle
}

type AgServer struct {
	common.Parent

	Server bootstrap.IServerStrap

	AgServerConf IAgServerConf

	// 处理location选择
	LocationHandle LocationHandle
}

// NewAgServer 创建代理服务端
// serverConf 不可为空
// locationHandle 可为空，则使用默认方式
// locations 定位配置，不可为空
func NewAgServer(parent interface{}, agServerConf IAgServerConf, locationHandle LocationHandle) *AgServer {
	if agServerConf == nil {
		err := "agServerConf is nil"
		logx.Error(err)
		panic(err)
	}

	b := &AgServer{AgServerConf: agServerConf}
	b.SetParent(parent)
	if locationHandle != nil {
		b.LocationHandle = locationHandle
	} else {
		// 使用默认location
		b.LocationHandle = defaultLocationHandle
	}
	return b
}

func (bg *AgServer) GetServer() bootstrap.IServerStrap {
	return bg.Server
}

func (bg *AgServer) GetAgServerConf() IAgServerConf {
	return bg.AgServerConf
}

func (bg *AgServer) GetLocationHandle() LocationHandle {
	return bg.LocationHandle
}

// LocationHandle 获取location，以便确认upstream的处理
type LocationHandle func(server IAgServer, pattern string, params ...interface{}) ILocationConf

// defaultLocationHandle 默认LocationHandle，使用随机分配算法
func defaultLocationHandle(server IAgServer, pattern string, params ...interface{}) ILocationConf {
	locations := server.GetAgServerConf().GetLocationConfs()
	aglc := DefaultLocation
	if locations != nil {
		if len(pattern) > 0 {
			lc, found := locations[pattern]
			if found {
				aglc = lc
			} else {
				return nil
			}
		}
	}
	return aglc
}

var DefaultLocation ILocationConf = NewLocationConf("", "", nil)
