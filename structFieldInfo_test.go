package zorm

import (
	"reflect"
	"testing"
)

func Test_structFieldInfo(t *testing.T) {
	type args struct {
		typeOf *reflect.Type
	}

	typeOf1 := reflect.TypeOf(struct {
		UserName string `column:"user_name"`
	}{})

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := structFieldInfo(tt.args.typeOf); (err != nil) != tt.wantErr {
				t.Errorf("structFieldInfo() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 获取缓存, 这里只测试 dbColumnNamePrefix
			info, err := getCacheStructFieldInfo(&typeOf1, dbColumnNamePrefix)
			if err != nil {
				t.Errorf("getCacheStructFieldInfo() error = %v", err)
			}

			// 转换类型
			mp, ok := (*info).(map[string]reflect.StructField)
			if !ok {
				t.Errorf("not of reflect.StructFile")
			}

			// 缓存的field是否和上面定义的一样
			for _, field := range mp {
				if field.Name != "UserName" {
					t.Errorf("get cache error, field.Name = %s, want = UserName", field.Name)
				}
			}
		})
	}
}
