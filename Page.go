package zorm

//Page 分页对象
type Page struct {
	//当前页码,从1开始
	PageNo int
	//每页多少条,默认20条
	PageSize int
	//数据总条数
	TotalCount int
	//总共多少页
	PageCount int
	//是否是第一页
	FirstPage bool
	//是否有上一页
	HasPrev bool
	//是否有下一页
	HasNext bool
	//是否是最后一页
	LastPage bool
}

//NewPage 创建Page对象
func NewPage() Page {
	page := Page{}
	page.PageNo = 1
	page.PageSize = 20
	return page
}

//setTotalCount 设置总条数,计算其他值
func (page *Page) setTotalCount(total int) {
	page.TotalCount = total
	page.PageCount = (page.TotalCount + page.PageSize - 1) / page.PageSize
	if page.PageNo >= page.PageCount {
		page.LastPage = true
	} else {
		page.HasNext = true
	}
	if page.PageNo > 1 {
		page.HasPrev = true
	} else {
		page.FirstPage = true
	}

}
