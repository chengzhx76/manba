package proxy

import (
	"manba/pkg/filter"
	"manba/pkg/util"
)

// PrepareFilter必须在过滤器链的第一个中，用于获取一些公共信息到上下文中，避免后续的过滤器执行重复的操作。

// PrepareFilter Must be in the first of the filter chain,
// used to get some public information into the context,
// to avoid subsequent filters to do duplicate things.
type PrepareFilter struct {
	filter.BaseFilter // 相当于 Java 的继承
}

func newPrepareFilter() filter.Filter {
	return &PrepareFilter{}
}

// Init init filter
func (f *PrepareFilter) Init(cfg string) error {
	return nil
}

// Name return name of this filter
func (f *PrepareFilter) Name() string {
	return FilterPrepare
}

// Pre execute before proxy
func (f *PrepareFilter) Pre(c filter.Context) (statusCode int, err error) {
	c.SetAttr(filter.AttrClientRealIP, util.ClientIP(c.OriginRequest()))
	return f.BaseFilter.Pre(c) // 执行基础过滤器，基础过滤器只返回 http 状态码 200
	// 相当于 Java 的 this.父类方法
}
