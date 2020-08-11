/*
 * Author:slive
 * DATE:2020/8/6
 */
package agservice

import (
	"gsfly/bootstrap"
	"time"
)

type AgServer interface {
	GetServer() bootstrap.ServerStrap

	GetServerConf() bootstrap.ServerConf

	GetLocationMap() map[string]AgLocation

	GetLocations() []AgLocation

	GetLocationHandle() LocationHandle
}

type BaseAgServer struct {
	Server     bootstrap.ServerStrap
	ServerConf bootstrap.ServerConf

	LocationMap map[string]AgLocation
	Locations   []AgLocation

	// 处理location选择
	LocationHandle LocationHandle
}

func NewBaseAgServer(serverConf bootstrap.ServerConf, locationHandle LocationHandle, locations ...AgLocation) *BaseAgServer {
	b := &BaseAgServer{ServerConf: serverConf}

	if locationHandle != nil {
		b.LocationHandle = locationHandle
	} else {
		// 使用默认location
		b.LocationHandle = defaultLocationHandle
	}

	llen := len(locations)
	if llen > 0 {
		b.LocationMap = make(map[string]AgLocation, llen)
		b.Locations = locations
		for _, v := range locations {
			b.LocationMap[v.GetPattern()] = v
		}
	}
	return b
}

func (bg *BaseAgServer) GetServer() bootstrap.ServerStrap {
	return bg.Server
}

func (bg *BaseAgServer) GetServerConf() bootstrap.ServerConf {
	return bg.ServerConf
}

func (bg *BaseAgServer) GetLocationMap() map[string]AgLocation {
	return bg.LocationMap
}

func (bg *BaseAgServer) GetLocations() []AgLocation {
	return bg.Locations
}

func (bg *BaseAgServer) GetLocationHandle() LocationHandle {
	return bg.LocationHandle
}

// LocationHandle 获取location，以便确认upstream的处理
type LocationHandle func(server AgServer, pattern string, params ...interface{}) AgLocation

// defaultLocationHandle 默认LocationHandle，使用随机分配算法
func defaultLocationHandle(server AgServer, pattern string, params ...interface{}) AgLocation {
	locations := server.GetLocations()
	if locations != nil {
		if len(pattern) > 0{
			return server.GetLocationMap()[pattern]
		} else {
			len := len(locations)
			if len > 0 {
				// 随机方式
				selectIndex := time.Now().Second() % len
				return locations[selectIndex]
			}
		}
	}
	return DefaultLocation
}

var DefaultLocation AgLocation = NewBaseAgLocation("", "")
