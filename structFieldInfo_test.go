package zorm

import (
	"reflect"
	"testing"

	models1 "gitee.com/chunanyong/zorm/mydir"
	models2 "gitee.com/chunanyong/zorm/myfoo"
)

func Test_structFieldInfo(t *testing.T) {
	type args struct {
		typeOf *reflect.Type
	}

	typeOf1 := reflect.TypeOf(models1.MyModel{})
	typeOf2 := reflect.TypeOf(models2.MyModel{})

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "",
			args: args{
				typeOf: &typeOf1,
			},
			wantErr: false,
		},
		{
			name: "",
			args: args{
				typeOf: &typeOf2,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := structFieldInfo(tt.args.typeOf); (err != nil) != tt.wantErr {
				t.Errorf("structFieldInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
