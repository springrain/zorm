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
	"strings"
	"testing"
)

func Test_FuncWrapFieldTagName_dialectFromConfig(t *testing.T) {
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

func Test_wrapUpdateEntityMapSQL_stableColumnOrder(t *testing.T) {
	ctx := context.Background()

	// Step 1: Create EntityMap and Set fields in specific order
	entity := NewEntityMap("t_user")
	entity.PkColumnName = "id"
	entity.Set("id", 1)
	entity.Set("status", 1)
	entity.Set("name", "test")
	entity.Set("age", 25)
	entity.Set("email", "test@example.com")

	// Step 2: Call wrapUpdateEntityMapSQL multiple times, verify column order is stable
	expectedOrder := []string{"status", "name", "age", "email"}

	for i := 0; i < 10; i++ {
		sqlstr, values, err := wrapUpdateEntityMapSQL(ctx, entity)
		if err != nil {
			t.Fatalf("wrapUpdateEntityMapSQL error: %v", err)
		}

		// Verify column order matches Set() call order
		for j, col := range expectedOrder {
			expected := col + "=?"
			idx := strings.Index(*sqlstr, expected)
			if idx == -1 {
				t.Errorf("iteration %d: SQL missing column %s, sql=%s", i, expected, *sqlstr)
			} else {
				// Verify columns appear in correct order
				if j > 0 {
					prevCol := expectedOrder[j-1] + "=?"
					prevIdx := strings.Index(*sqlstr, prevCol)
					if prevIdx >= idx {
						t.Errorf("iteration %d: column %s should appear before %s, sql=%s", i, prevCol, expected, *sqlstr)
					}
				}
			}
		}

		// Verify values order (excluding primary key which is last)
		if len(*values) != len(expectedOrder)+1 {
			t.Errorf("iteration %d: expected %d values, got %d", i, len(expectedOrder)+1, len(*values))
		}

		// Verify primary key is WHERE clause value
		pkValue := (*values)[len(*values)-1]
		if pkValue != 1 {
			t.Errorf("iteration %d: expected pk value 1, got %v", i, pkValue)
		}

		// Verify value order matches column order
		for j, col := range expectedOrder {
			expectedVal := entity.GetDBFieldMap()[col]
			if (*values)[j] != expectedVal {
				t.Errorf("iteration %d: column %s expected value %v, got %v", i, col, expectedVal, (*values)[j])
			}
		}
	}

	// Step 3: Verify generated SQL has correct format
	sqlstr, _, _ := wrapUpdateEntityMapSQL(ctx, entity)
	expectedSQL := "UPDATE t_user SET status=?,name=?,age=?,email=? WHERE id=?"
	if *sqlstr != expectedSQL {
		t.Errorf("SQL mismatch.\nExpected: %s\nGot:      %s", expectedSQL, *sqlstr)
	}
}

func Test_wrapUpdateEntityMapSQL_stableColumnOrder_multiple(t *testing.T) {
	ctx := context.Background()

	// Run multiple times to check for deterministic behavior
	for run := 0; run < 100; run++ {
		entity := NewEntityMap("t_user")
		entity.PkColumnName = "id"
		entity.Set("zebra", 1)
		entity.Set("apple", 2)
		entity.Set("mango", 3)
		entity.Set("banana", 4)
		entity.Set("id", 999)

		sqlstr, values, err := wrapUpdateEntityMapSQL(ctx, entity)
		if err != nil {
			t.Fatalf("wrapUpdateEntityMapSQL error: %v", err)
		}

		// Must match the Set() call order
		expectedSQL := "UPDATE t_user SET zebra=?,apple=?,mango=?,banana=? WHERE id=?"
		if *sqlstr != expectedSQL {
			t.Errorf("run %d: expected SQL %q, got %q", run, expectedSQL, *sqlstr)
		}

		// Values must follow the same order: zebra, apple, mango, banana, id
		if len(*values) != 5 {
			t.Fatalf("run %d: expected 5 values, got %d", run, len(*values))
		}
		expectedVals := []interface{}{1, 2, 3, 4, 999}
		for vi, ev := range expectedVals {
			if (*values)[vi] != ev {
				t.Errorf("run %d: values[%d] = %v, want %v", run, vi, (*values)[vi], ev)
			}
		}
	}
}

func Test_FuncWrapFieldTagName(t *testing.T) {
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

				return fmt.Sprintf("`%s`", colName)
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

func Test_firstOneWord(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		index     int
		wantWord  string
		wantStart int
		wantEnd   int
		wantErr   bool
	}{
		{
			name:     "normal word",
			input:    "SELECT * FROM users",
			index:    0,
			wantWord: "SELECT",
			wantStart: 0,
			wantErr:  false,
		},
		{
			name:     "skip leading spaces, start points to word after spaces",
			input:    "  FROM users",
			index:    0,
			wantWord: "FROM",
			wantStart: 2,
			wantErr:  false,
		},
		{
			name:     "skip leading parentheses, start points to word",
			input:    "(SELECT)",
			index:    0,
			wantWord: "SELECT",
			wantStart: 1,
			wantErr:  false,
		},
		{
			name:     "mixed spaces and parentheses",
			input:    " ( ( FROM ",
			index:    0,
			wantWord: "FROM",
			wantStart: 5,
			wantErr:  false,
		},
		{
			name:      "index out of range",
			input:     "SELECT",
			index:     10,
			wantWord:  "",
			wantStart: -1,
			wantEnd:   -1,
			wantErr:   true,
		},
		{
			name:      "negative index",
			input:     "SELECT",
			index:     -1,
			wantWord:  "",
			wantStart: -1,
			wantEnd:   -1,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWord, gotStart, gotEnd, err := firstOneWord(tt.index, []byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("firstOneWord() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotWord != tt.wantWord {
				t.Errorf("firstOneWord() word = %v, want %v", gotWord, tt.wantWord)
			}
			if tt.wantStart != -1 && gotStart != tt.wantStart {
				t.Errorf("firstOneWord() start = %v, want %v", gotStart, tt.wantStart)
			}
			if tt.wantErr {
				return
			}
			if gotWord != "" && gotEnd < gotStart {
				t.Errorf("firstOneWord() end %v should >= start %v", gotEnd, gotStart)
			}
		})
	}
}

func Test_EntityMap_Set_DuplicateKey(t *testing.T) {
	entity := NewEntityMap("t_test")
	entity.PkColumnName = "id"

	entity.Set("name", "first")
	// Set same key again, should only record key once
	entity.Set("name", "second")

	keys := entity.GetDBFieldMapKey()
	count := 0
	for _, k := range keys {
		if k == "name" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("duplicate key in dbFieldMapKey, count=%d, want 1", count)
	}

	if entity.GetDBFieldMap()["name"] != "second" {
		t.Errorf("value not updated, got %v, want 'second'", entity.GetDBFieldMap()["name"])
	}
}

func Test_EntityMap_JSON_MarshalUnmarshal(t *testing.T) {
	entity := NewEntityMap("t_user")
	entity.PkColumnName = "id"
	entity.Set("id", 100)
	entity.Set("name", "test")
	entity.Set("age", 25)

	data, err := entity.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}

	newEntity := NewEntityMap("t_user")
	if err := newEntity.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}

	if newEntity.GetTableName() != entity.GetTableName() {
		t.Errorf("tableName mismatch: got %s, want %s", newEntity.GetTableName(), entity.GetTableName())
	}

	if newEntity.PkColumnName != entity.PkColumnName {
		t.Errorf("PkColumnName mismatch: got %s, want %s", newEntity.PkColumnName, entity.PkColumnName)
	}

	for _, key := range entity.GetDBFieldMapKey() {
		newVal := newEntity.GetDBFieldMap()[key]
		oldVal := entity.GetDBFieldMap()[key]
		if fmt.Sprintf("%v", newVal) != fmt.Sprintf("%v", oldVal) {
			t.Errorf("value mismatch for key %s: got %v (type %T), want %v (type %T)",
				key, newVal, newVal, oldVal, oldVal)
		}
	}
}

func Test_Page_setTotalCount(t *testing.T) {
	tests := []struct {
		name     string
		page     *Page
		total    int
		wantPage *Page
	}{
		{
			name:     "exact multiple, page 1 of 3",
			page:     &Page{PageNo: 1, PageSize: 10},
			total:    30,
			wantPage: &Page{TotalCount: 30, PageCount: 3, FirstPage: true, HasPrev: false, HasNext: true, LastPage: false},
		},
		{
			name:     "partial page, page 1 of 3",
			page:     &Page{PageNo: 1, PageSize: 10},
			total:    25,
			wantPage: &Page{TotalCount: 25, PageCount: 3, FirstPage: true, HasPrev: false, HasNext: true, LastPage: false},
		},
		{
			name:     "middle page, page 2 of 3",
			page:     &Page{PageNo: 2, PageSize: 10},
			total:    25,
			wantPage: &Page{TotalCount: 25, PageCount: 3, FirstPage: false, HasPrev: true, HasNext: true, LastPage: false},
		},
		{
			name:     "last page, page 3 of 3",
			page:     &Page{PageNo: 3, PageSize: 10},
			total:    25,
			wantPage: &Page{TotalCount: 25, PageCount: 3, FirstPage: false, HasPrev: true, HasNext: false, LastPage: true},
		},
		{
			name:     "zero total",
			page:     &Page{PageNo: 1, PageSize: 10},
			total:    0,
			wantPage: &Page{TotalCount: 0, PageCount: 0, FirstPage: true, HasPrev: false, HasNext: false, LastPage: true},
		},
		{
			name:     "single item, one page",
			page:     &Page{PageNo: 1, PageSize: 10},
			total:    1,
			wantPage: &Page{TotalCount: 1, PageCount: 1, FirstPage: true, HasPrev: false, HasNext: false, LastPage: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.page.setTotalCount(tt.total)
			if tt.page.TotalCount != tt.wantPage.TotalCount {
				t.Errorf("TotalCount = %v, want %v", tt.page.TotalCount, tt.wantPage.TotalCount)
			}
			if tt.page.PageCount != tt.wantPage.PageCount {
				t.Errorf("PageCount = %v, want %v", tt.page.PageCount, tt.wantPage.PageCount)
			}
			if tt.page.FirstPage != tt.wantPage.FirstPage {
				t.Errorf("FirstPage = %v, want %v", tt.page.FirstPage, tt.wantPage.FirstPage)
			}
			if tt.page.HasPrev != tt.wantPage.HasPrev {
				t.Errorf("HasPrev = %v, want %v", tt.page.HasPrev, tt.wantPage.HasPrev)
			}
			if tt.page.HasNext != tt.wantPage.HasNext {
				t.Errorf("HasNext = %v, want %v", tt.page.HasNext, tt.wantPage.HasNext)
			}
			if tt.page.LastPage != tt.wantPage.LastPage {
				t.Errorf("LastPage = %v, want %v", tt.page.LastPage, tt.wantPage.LastPage)
			}
		})
	}
}

func Test_NewPage_defaults(t *testing.T) {
	page := NewPage()
	if page.PageNo != 1 {
		t.Errorf("PageNo = %v, want 1", page.PageNo)
	}
	if page.PageSize != 20 {
		t.Errorf("PageSize = %v, want 20", page.PageSize)
	}
}

func Test_wrapUpdateEntityMapSQL_emptyDBFieldMap(t *testing.T) {
	ctx := context.Background()
	entity := NewEntityMap("t_test")
	entity.PkColumnName = "id"

	_, _, err := wrapUpdateEntityMapSQL(ctx, entity)
	if err == nil {
		t.Error("expected error for empty dbFieldMap, got nil")
	}
}

func Test_wrapUpdateEntityMapSQL_PKSetLast(t *testing.T) {
	ctx := context.Background()
	entity := NewEntityMap("t_order")
	entity.PkColumnName = "order_id"

	entity.Set("status", "paid")
	entity.Set("amount", 99.9)
	entity.Set("order_id", "ORD001")

	sqlstr, values, err := wrapUpdateEntityMapSQL(ctx, entity)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	expectedSQL := "UPDATE t_order SET status=?,amount=? WHERE order_id=?"
	if *sqlstr != expectedSQL {
		t.Errorf("SQL = %q, want %q", *sqlstr, expectedSQL)
	}

	if len(*values) != 3 {
		t.Fatalf("values len = %d, want 3", len(*values))
	}
	if (*values)[2] != "ORD001" {
		t.Errorf("PK value = %v, want 'ORD001'", (*values)[2])
	}
}

func Test_wrapUpdateEntityMapSQL_PKOnly(t *testing.T) {
	ctx := context.Background()
	entity := NewEntityMap("t_user")
	entity.PkColumnName = "id"
	entity.Set("id", 1)

	sqlstr, values, err := wrapUpdateEntityMapSQL(ctx, entity)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	expectedSQL := "UPDATE t_user SET  WHERE id=?"
	if *sqlstr != expectedSQL {
		t.Errorf("SQL = %q, want %q", *sqlstr, expectedSQL)
	}

	if len(*values) != 1 {
		t.Errorf("values len = %d, want 1", len(*values))
	}
}

func Test_wrapUpdateEntityMapSQL_PKFirstInSet(t *testing.T) {
	ctx := context.Background()
	entity := NewEntityMap("t_user")
	entity.PkColumnName = "id"

	// PK set first, then other fields
	entity.Set("id", 42)
	entity.Set("name", "alice")
	entity.Set("age", 30)

	sqlstr, values, err := wrapUpdateEntityMapSQL(ctx, entity)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	// PK should NOT be in SET clause, only in WHERE
	expectedSQL := "UPDATE t_user SET name=?,age=? WHERE id=?"
	if *sqlstr != expectedSQL {
		t.Errorf("SQL = %q, want %q", *sqlstr, expectedSQL)
	}

	// Values: name, age, id(pk)
	if len(*values) != 3 {
		t.Fatalf("values len = %d, want 3", len(*values))
	}
	if (*values)[0] != "alice" {
		t.Errorf("values[0] = %v, want 'alice'", (*values)[0])
	}
	if (*values)[1] != 30 {
		t.Errorf("values[1] = %v, want 30", (*values)[1])
	}
	if (*values)[2] != 42 {
		t.Errorf("values[2] = %v, want 42", (*values)[2])
	}
}
