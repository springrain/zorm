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
	"reflect"
	"testing"
)

// 测试用的简化配置
type testDataSourceConfig struct {
	Dialect string
}

// testEntity 测试用的实体结构体
type testEntity struct {
	// ID 主键
	ID string `column:"id"`

	// UserName 用户名
	UserName string `column:"user_name"`

	// Age 年龄
	Age int `column:"age"`

	// Email 邮箱
	Email string `column:"email"`

	// IsActive 是否激活
	IsActive bool `column:"is_active"`
}

// GetTableName 获取表名称
func (entity *testEntity) GetTableName() string {
	return "test_table"
}

// GetPKColumnName 获取主键字段名称
func (entity *testEntity) GetPKColumnName() string {
	return "id"
}

// GetPkSequence 主键序列
func (entity *testEntity) GetPkSequence() string {
	return ""
}

// GetDefaultValue 获取默认值
func (entity *testEntity) GetDefaultValue() map[string]interface{} {
	return nil
}

func Test_structFieldCache(t *testing.T) {
	ctx := context.Background()

	// 测试结构体类型
	type testStruct struct {
		ID       string `column:"id"`
		UserName string `column:"user_name"`
		Age      int    `column:"age"`
		Email    string `column:"email"`
		IsActive bool   `column:"is_active"`
	}

	// 获取结构体类型
	typeOf := reflect.TypeOf(testStruct{})

	t.Run("测试结构体字段缓存", func(t *testing.T) {
		// 由于我们不能直接调用 getStructTypeOfCache（需要 DataSourceConfig 类型）
		// 我们测试字段解析的逻辑
		entityCache := &entityStructCache{
			fields:    make([]*fieldColumnCache, 0),
			fieldMap:  make(map[string]*fieldColumnCache),
			columns:   make([]*fieldColumnCache, 0),
			columnMap: make(map[string]*fieldColumnCache),
		}

		// 手动处理字段
		fieldNum := typeOf.NumField()
		for i := 0; i < fieldNum; i++ {
			field := typeOf.Field(i)
			funcCreateEntityStructCache(ctx, entityCache, field)
		}

		// 检查字段数量
		expectedFields := []string{"id", "user_name", "age", "email", "is_active"}
		if len(entityCache.columns) != len(expectedFields) {
			t.Errorf("entityCache.columns length = %d, want %d", len(entityCache.columns), len(expectedFields))
		}

		// 检查字段映射
		for _, fieldName := range expectedFields {
			if _, ok := entityCache.columnMap[fieldName]; !ok {
				t.Errorf("field %s not found in columnMap", fieldName)
			}
		}

		// 检查字段名映射
		expectedFieldNames := []string{"id", "username", "age", "email", "isactive"}
		for _, fieldName := range expectedFieldNames {
			if _, ok := entityCache.fieldMap[fieldName]; !ok {
				t.Errorf("field name %s not found in fieldMap", fieldName)
			}
		}
	})

	t.Run("测试字段标签解析", func(t *testing.T) {
		// 测试不同的标签格式
		type tagTestStruct struct {
			Field1 string `column:"field_1"`
			Field2 string `column:"field2"`
			Field3 string // 没有column标签
			Field4 string `column:""` // 空标签
		}

		typeOf2 := reflect.TypeOf(tagTestStruct{})
		entityCache := &entityStructCache{
			fields:    make([]*fieldColumnCache, 0),
			fieldMap:  make(map[string]*fieldColumnCache),
			columns:   make([]*fieldColumnCache, 0),
			columnMap: make(map[string]*fieldColumnCache),
		}

		fieldNum := typeOf2.NumField()
		for i := 0; i < fieldNum; i++ {
			field := typeOf2.Field(i)
			funcCreateEntityStructCache(ctx, entityCache, field)
		}

		// 只有有column标签的字段应该被添加到columns
		if len(entityCache.columns) != 2 {
			t.Errorf("entityCache.columns length = %d, want 2 (only fields with column tag)", len(entityCache.columns))
		}

		// 检查具体的字段
		if _, ok := entityCache.columnMap["field_1"]; !ok {
			t.Error("field_1 not found in columnMap")
		}

		if _, ok := entityCache.columnMap["field2"]; !ok {
			t.Error("field2 not found in columnMap")
		}

		// Field3和Field4不应该在columnMap中
		if _, ok := entityCache.columnMap["field3"]; ok {
			t.Error("field3 should not be in columnMap (no column tag)")
		}

		if _, ok := entityCache.columnMap["field4"]; ok {
			t.Error("field4 should not be in columnMap (empty column tag)")
		}
	})
}
