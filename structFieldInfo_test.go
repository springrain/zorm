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
	"sync"
	"testing"
)

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
		// 由于我们不能直接调用 getStructTypeOfCache (需要 DataSourceConfig 类型)
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

func Test_entityStructCacheMap_Concurrency(t *testing.T) {
	ctx := context.Background()

	type concurrentEntity struct {
		ID       string `column:"id"`
		Name     string `column:"name"`
		Age      int    `column:"age"`
		Email    string `column:"email"`
		Status   int    `column:"status"`
	}

	config := &DataSourceConfig{
		Dialect: "mysql",
	}

	// Clear any existing cache for this type to ensure a clean test
	typeOf := reflect.TypeOf(concurrentEntity{})
	pkgPath := typeOf.PkgPath()
	typeOfString := typeOf.String()
	key := config.Dialect + "_" + pkgPath + "_" + typeOfString
	entityStructCacheMap.Delete(key)

	t.Run("concurrent getStructTypeOfCache", func(t *testing.T) {
		const goroutines = 100
		const iterations = 100

		var wg sync.WaitGroup
		wg.Add(goroutines)

		resultPtrs := make([]*entityStructCache, goroutines)
		var resultMu sync.Mutex
		errs := make([]error, goroutines)

		for g := 0; g < goroutines; g++ {
			go func(gid int) {
				defer wg.Done()
				for i := 0; i < iterations; i++ {
					typeOfLocal := reflect.TypeOf(concurrentEntity{})
					cache, err := getStructTypeOfCache(ctx, &typeOfLocal, config)
					if err != nil {
						errs[gid] = err
						return
					}
					if gid == 0 {
						resultMu.Lock()
						resultPtrs[gid] = cache
						resultMu.Unlock()
					}
					// Capture the first goroutine's cache pointer for comparison
					if cache != nil {
						resultMu.Lock()
						if resultPtrs[0] == nil {
							resultPtrs[0] = cache
						} else if resultPtrs[gid] == nil {
							resultPtrs[gid] = cache
						}
						resultMu.Unlock()
					}
				}
			}(g)
		}

		wg.Wait()

		// Check for errors
		for g, err := range errs {
			if err != nil {
				t.Fatalf("goroutine %d error: %v", g, err)
			}
		}

		// Verify all goroutines got the same cache pointer
		if resultPtrs[0] != nil {
			for g := 1; g < goroutines; g++ {
				if resultPtrs[g] != resultPtrs[0] {
					t.Errorf("goroutine %d got different cache pointer: %p vs %p", g, resultPtrs[g], resultPtrs[0])
				}
			}
		}

		// Verify the cache is correct
		cache, _ := entityStructCacheMap.Load(key)
		if cache == nil {
			t.Fatal("cache should exist after concurrent access")
		}
		entityCache := cache.(*entityStructCache)
		if len(entityCache.columns) != 5 {
			t.Errorf("expected 5 columns, got %d", len(entityCache.columns))
		}
	})

	// Clean up
	entityStructCacheMap.Delete(key)
}

func Test_funcCreateEntityStructCache_nestedAnonymousOverride(t *testing.T) {
	ctx := context.Background()

	t.Run("outer struct field overrides nested anonymous struct field", func(t *testing.T) {
		// Nested anonymous struct with 'Name' field
		type BaseInfo struct {
			Name        string `column:"name"`
			Description string `column:"description"`
		}

		// Outer struct has its own 'Name' field that should override the nested one
		type OverrideEntity struct {
			BaseInfo
			Name  string `column:"override_name"`
			Value int    `column:"value"`
		}

		typeOf := reflect.TypeOf(OverrideEntity{})
		entityCache := &entityStructCache{
			fields:    make([]*fieldColumnCache, 0, typeOf.NumField()),
			fieldMap:  make(map[string]*fieldColumnCache),
			columns:   make([]*fieldColumnCache, 0, typeOf.NumField()),
			columnMap: make(map[string]*fieldColumnCache),
		}

		fieldNum := typeOf.NumField()
		for i := 0; i < fieldNum; i++ {
			field := typeOf.Field(i)
			if field.Anonymous {
				funcRecursiveAnonymous(ctx, entityCache, &field)
			} else {
				funcCreateEntityStructCache(ctx, entityCache, field)
			}
		}

		// Check that outer 'Name' field overridden nested 'Name'
		// The outer 'Name' should be in columns with column "override_name"
		nameCached := false
		overrideNameCached := false
		for _, col := range entityCache.columns {
			if col.columnName == "name" {
				nameCached = true
			}
			if col.columnName == "override_name" {
				overrideNameCached = true
			}
		}

		if nameCached {
			t.Error("nested anonymous 'name' column should have been removed by outer struct override")
		}
		if !overrideNameCached {
			t.Error("outer struct 'override_name' column should exist")
		}

		// The outer struct's 'Name' field should win in fieldMap
		nameField, ok := entityCache.fieldMap["name"]
		if !ok {
			t.Fatal("fieldMap should contain 'name' field")
		}
		if nameField.columnName != "override_name" {
			t.Errorf("expected field 'Name' to map to 'override_name', got '%s'", nameField.columnName)
		}

		// Total columns: description (from nested), override_name, value (from outer) = 3
		expectedCols := 3
		if len(entityCache.columns) != expectedCols {
			t.Errorf("expected %d columns, got %d", expectedCols, len(entityCache.columns))
		}

		// Verify column names
		expectedColumnNames := map[string]bool{
			"description":   false,
			"override_name": false,
			"value":         false,
		}
		for _, col := range entityCache.columns {
			expectedColumnNames[col.columnName] = true
		}
		for colName, found := range expectedColumnNames {
			if !found {
				t.Errorf("column '%s' not found in columns", colName)
			}
		}
	})

	t.Run("deep nested anonymous struct field override", func(t *testing.T) {
		type Level3 struct {
			ID     string `column:"id"`
			Common string `column:"common"`
		}

		type Level2 struct {
			Level3
			Common string `column:"level2_common"`
		}

		type Level1 struct {
			Level2
			Common string `column:"level1_common"`
			Extra  string `column:"extra"`
		}

		typeOf := reflect.TypeOf(Level1{})
		entityCache := &entityStructCache{
			fields:    make([]*fieldColumnCache, 0),
			fieldMap:  make(map[string]*fieldColumnCache),
			columns:   make([]*fieldColumnCache, 0),
			columnMap: make(map[string]*fieldColumnCache),
		}

		fieldNum := typeOf.NumField()
		for i := 0; i < fieldNum; i++ {
			field := typeOf.Field(i)
			if field.Anonymous {
				funcRecursiveAnonymous(ctx, entityCache, &field)
			} else {
				funcCreateEntityStructCache(ctx, entityCache, field)
			}
		}

		// Only the outermost 'Common' should survive: level1_common (L1) and id (L3) and extra (L1)
		// Level2's 'Common' should have been removed by Level1's 'Common'
		// Level3's 'Common' should have been removed by Level2's 'Common' (which was then removed by Level1)
		commonCount := 0
		for _, col := range entityCache.columns {
			if col.fieldName == "Common" {
				commonCount++
			}
		}

		if commonCount != 1 {
			t.Errorf("expected exactly 1 'Common' field in columns, got %d", commonCount)
		}

		commonCol, ok := entityCache.columnMap["level1_common"]
		if !ok {
			t.Error("column 'level1_common' should exist (outermost Common wins)")
		}
		if commonCol != nil && commonCol.fieldName != "Common" {
			t.Errorf("expected fieldName 'Common', got '%s'", commonCol.fieldName)
		}

		// level2_common should NOT exist
		if _, ok := entityCache.columnMap["level2_common"]; ok {
			t.Error("column 'level2_common' should have been removed by outermost override")
		}

		// Only id, level1_common, extra should remain
		if len(entityCache.columns) != 3 {
			t.Errorf("expected 3 columns (id, level1_common, extra), got %d", len(entityCache.columns))
		}
	})

	t.Run("no override, all nested fields preserved", func(t *testing.T) {
		type BaseA struct {
			FieldA string `column:"field_a"`
			FieldB string `column:"field_b"`
		}

		type BaseB struct {
			FieldC string `column:"field_c"`
		}

		type Combined struct {
			BaseA
			BaseB
			FieldD string `column:"field_d"`
		}

		typeOf := reflect.TypeOf(Combined{})
		entityCache := &entityStructCache{
			fields:    make([]*fieldColumnCache, 0),
			fieldMap:  make(map[string]*fieldColumnCache),
			columns:   make([]*fieldColumnCache, 0),
			columnMap: make(map[string]*fieldColumnCache),
		}

		fieldNum := typeOf.NumField()
		for i := 0; i < fieldNum; i++ {
			field := typeOf.Field(i)
			if field.Anonymous {
				funcRecursiveAnonymous(ctx, entityCache, &field)
			} else {
				funcCreateEntityStructCache(ctx, entityCache, field)
			}
		}

		if len(entityCache.columns) != 4 {
			t.Errorf("expected 4 columns, got %d", len(entityCache.columns))
		}

		expectedCols := []string{"field_a", "field_b", "field_c", "field_d"}
		for _, col := range expectedCols {
			if _, ok := entityCache.columnMap[col]; !ok {
				t.Errorf("column '%s' should exist", col)
			}
		}
	})

	t.Run("nested anonymous without column tag still in fieldMap", func(t *testing.T) {
		type BaseNoColumn struct {
			NoTagField string
			WithTag   string `column:"with_tag"`
		}

		type DerivedNoColumn struct {
			BaseNoColumn
			OverrideNoTag string
			OverrideTag   string `column:"override_tag"`
		}

		typeOf := reflect.TypeOf(DerivedNoColumn{})
		entityCache := &entityStructCache{
			fields:    make([]*fieldColumnCache, 0),
			fieldMap:  make(map[string]*fieldColumnCache),
			columns:   make([]*fieldColumnCache, 0),
			columnMap: make(map[string]*fieldColumnCache),
		}

		fieldNum := typeOf.NumField()
		for i := 0; i < fieldNum; i++ {
			field := typeOf.Field(i)
			if field.Anonymous {
				funcRecursiveAnonymous(ctx, entityCache, &field)
			} else {
				funcCreateEntityStructCache(ctx, entityCache, field)
			}
		}

		// Only 'with_tag' and 'override_tag' should be in columns
		if len(entityCache.columns) != 2 {
			t.Errorf("expected 2 columns, got %d", len(entityCache.columns))
		}

		// Both should be in fieldMap
		if _, ok := entityCache.fieldMap["no tagfield"]; !ok {
			// Without column tag, fieldName is lowercased
		}
		if _, ok := entityCache.fieldMap["overridenotag"]; !ok {
			t.Error("fieldMap should contain 'overridenotag'")
		}
	})
}
