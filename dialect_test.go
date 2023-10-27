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
		fn   func(*reflect.StructField, string) string
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
			fn: func(field *reflect.StructField, colName string) string {
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
			fn: func(field *reflect.StructField, colName string) string {
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
			fn: func(field *reflect.StructField, colName string) string {
				return colName
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
