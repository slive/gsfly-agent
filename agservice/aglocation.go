/*
 * Author:slive
 * DATE:2020/8/6
 */
package agservice

type AgLocation interface {
	GetPattern() string

	GetUpstreamName() string
}

type BaseAgLocation struct {
	Pattern      string
	UpstreamName string
}

func NewBaseAgLocation(pattern string, upstreamName string) AgLocation {
	b := &BaseAgLocation{
		Pattern:      pattern,
		UpstreamName: upstreamName,
	}
	return b
}

func (agl *BaseAgLocation) GetPattern() string {
	return agl.Pattern
}

func (agl *BaseAgLocation) GetUpstreamName() string {
	return agl.UpstreamName
}
