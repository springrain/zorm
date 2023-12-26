/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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
