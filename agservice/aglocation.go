/*
 * Author:slive
 * DATE:2020/8/6
 */
package agservice

// AgLocation 代理定位接口， 通过location定位到对应的route
type AgLocation interface {
	GetPattern() string

	GetRouteName() string
}

type BaseAgLocation struct {
	Pattern   string
	RouteName string
}

func NewBaseAgLocation(pattern string, routeName string) AgLocation {
	b := &BaseAgLocation{
		Pattern:   pattern,
		RouteName: routeName,
	}
	return b
}

// GetPattern 匹配规则，可支持模糊匹配
func (agl *BaseAgLocation) GetPattern() string {
	return agl.Pattern
}

// GetRouteName 获取到的route名称
func (agl *BaseAgLocation) GetRouteName() string {
	return agl.RouteName
}
