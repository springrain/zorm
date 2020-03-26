package zorm

import (
	"fmt"
	"testing"
)

func TestPage(t *testing.T) {
	page := NewPage()
	page.PageNo = 3
	sqlstr := "select id,name from t_user where id=? and name=? order by id asc"
	ss, _ := wrapPageSQL("postgresql", sqlstr, page)
	fmt.Println(ss)
}
