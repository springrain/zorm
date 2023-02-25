package zorm

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_getFieldTagName(t *testing.T) {
	type args struct {
		field *reflect.StructField
	}
	tests := []struct {
		name string
		args args
		fn   func(string) string
		want string
	}{
		{
			name: "test `",
			args: args{
				field: &reflect.StructField{
					Name: "Described",
					Tag:  `column:"described" json:"desc,omitempty"`,
				},
			},
			fn: func(colName string) string {
				return fmt.Sprintf("`%s`", colName)
			},
			want: "`described`",
		},
		{
			name: "test '",
			args: args{
				field: &reflect.StructField{
					Name: "Described",
					Tag:  `column:"described" json:"desc,omitempty"`,
				},
			},
			fn: func(colName string) string {
				return fmt.Sprintf("'%s'", colName)
			},
			want: "'described'",
		},
		{
			name: "test empty",
			args: args{
				field: &reflect.StructField{
					Name: "Described",
					Tag:  `column:"described" json:"desc,omitempty"`,
				},
			},
			fn: func(colName string) string {
				return fmt.Sprintf("%s", colName)
			},
			want: "described",
		},
		{
			name: "test default",
			args: args{
				field: &reflect.StructField{
					Name: "Described",
					Tag:  `column:"described" json:"desc,omitempty"`,
				},
			},
			fn:   nil,
			want: "described",
		},
	}
	tagMap := make(map[string]string)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fn != nil {
				FuncWrapFieldTagName = tt.fn
			}
			tagMap[tt.args.field.Name] = tt.args.field.Tag.Get("column")
			if got := getFieldTagName(tt.args.field, &tagMap); got != tt.want {
				t.Errorf("getFieldTagName() = %v, want %v", got, tt.want)
			}
		})
	}
}
