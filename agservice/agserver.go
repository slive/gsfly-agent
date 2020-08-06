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
	GetServer() bootstrap.Server

	GetServerConf() bootstrap.ServerConf

	GetLocationMap() map[string]AgLocation

	GetLocations() []AgLocation

	GetHandleLocation() HandleLocation
}

type BaseAgServer struct {
	Server     bootstrap.Server
	ServerConf bootstrap.ServerConf

	LocationMap map[string]AgLocation
	Locations   []AgLocation
	// 处理location选择
	HandleLocation HandleLocation
}

func NewBaseAgServer(serverConf bootstrap.ServerConf, handleLocation HandleLocation, locations... AgLocation) *BaseAgServer {
	b := &BaseAgServer{ServerConf: serverConf}

	if handleLocation != nil {
		b.HandleLocation = handleLocation
	} else {
		// 使用默认location
		b.HandleLocation = defaultHandleLocation
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

func (bg *BaseAgServer) GetServer() bootstrap.Server {
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

func (bg *BaseAgServer) GetHandleLocation() HandleLocation {
	return bg.HandleLocation
}

// HandleLocation 获取location，以便确认upstream的处理
type HandleLocation func(server AgServer, params ...interface{}) AgLocation

func defaultHandleLocation(server AgServer, params ...interface{}) AgLocation {
	locations := server.GetLocations()
	if locations != nil {
		len := len(locations)
		if len > 0 {
			// 随机方式
			selectIndex := time.Now().Second() % len
			return locations[selectIndex]
		}
	}
	return DefaultLocation
}

var DefaultLocation AgLocation = NewBaseAgLocation("", "")
