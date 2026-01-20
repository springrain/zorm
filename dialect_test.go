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
	"context"
	"fmt"
	"reflect"
	"testing"
)

func Test_getFiieldTagName_dialectFromConfig(t *testing.T) {
	type args struct {
		dialect string
		field   *reflect.StructField
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test mysql dialect from config",
			args: args{
				dialect: "mysql",
				field: &reflect.StructField{
					Name: "Described",
					Tag:  `column:"described" json:"desc,omitempty"`,
				},
			},
			want: "`described`",
		},
		{
			name: "test postgres dialect from config",
			args: args{
				dialect: "postgres",
				field: &reflect.StructField{
					Name: "Described",
					Tag:  `column:"described" json:"desc,omitempty"`,
				},
			},
			want: `"described"`,
		},
		{
			name: "test default",
			args: args{
				field: &reflect.StructField{
					Name: "Described",
					Tag:  `column:"described" json:"desc,omitempty"`,
				},
			},
			want: "described",
		},
	}

	// 全局包裹函数
	FuncWrapFieldTagName = func(ctx context.Context, field *reflect.StructField, colName string) string {
		config, err := GetContextDataSourceConfig(ctx)
		if err != nil {
			return ""
		}

		if config != nil && config.Dialect != "" {
			switch config.Dialect {
			case "mysql":
				return fmt.Sprintf("`%s`", colName)
			case "postgres":
				return fmt.Sprintf(`"%s"`, colName)
				// case ...
			}
		}

		return colName
	}

	emptyCtx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			defaultDao = &DBDao{
				config: &DataSourceConfig{
					Dialect: tt.args.dialect,
				},
				dataSource: &dataSource{},
			}
			dbConnection, errGetDBConnection := defaultDao.newDBConnection()
			if errGetDBConnection != nil {
				t.Fatalf("errGetDBConnection: %v", errGetDBConnection)
			}

			// 把dbConnection放入context
			ctx := context.WithValue(emptyCtx, contextDBConnectionValueKey, dbConnection)

			if got := FuncWrapFieldTagName(ctx, tt.args.field, tt.args.field.Tag.Get("column")); got != tt.want {
				t.Errorf("FuncWrapFieldTagName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getFieldTagName(t *testing.T) {
	type args struct {
		field *reflect.StructField
	}
	tests := []struct {
		name string
		args args
		fn   func(context.Context, *reflect.StructField, string) string
		want string
	}{
		{
			name: "test dialect from config",
			args: args{
				field: &reflect.StructField{
					Name: "Described",
					Tag:  `column:"described" json:"desc,omitempty"`,
				},
			},
			fn: func(ctx context.Context, field *reflect.StructField, colName string) string {
				config, err := GetContextDataSourceConfig(ctx)
				if err != nil {
					return ""
				}

				if config != nil && config.Dialect != "" {
					switch config.Dialect {
					case "mysql":
						return fmt.Sprintf("`%s`", colName)
					case "postgres":
						return fmt.Sprintf(`"%s"`, colName)
						// case ...
					}
				}

				return colName
			},
			want: "`described`",
		},
		{
			name: "test `",
			args: args{
				field: &reflect.StructField{
					Name: "Described",
					Tag:  `column:"described" json:"desc,omitempty"`,
				},
			},
			fn: func(ctx context.Context, field *reflect.StructField, colName string) string {
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
			fn: func(ctx context.Context, field *reflect.StructField, colName string) string {
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
			fn: func(ctx context.Context, field *reflect.StructField, colName string) string {
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

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fn != nil {
				FuncWrapFieldTagName = tt.fn
			}
			if got := FuncWrapFieldTagName(ctx, tt.args.field, tt.args.field.Tag.Get("column")); got != tt.want {
				t.Errorf("FuncWrapFieldTagName() = %v, want %v", got, tt.want)
			}
		})
	}
}
