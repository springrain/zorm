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
 */

package zorm

import (
	"context"
	"reflect"
	"testing"
)

func TestIoTDBDialectRegistered(t *testing.T) {
	if driverName := dbDialectMap["iotdb"]; driverName != "iotdb" {
		t.Fatalf("IoTDB driver name = %q, want %q", driverName, "iotdb")
	}
}

func TestIoTDBPagination(t *testing.T) {
	finder := NewSelectFinder("root.sg.device1", "temperature")
	page := &Page{PageNo: 3, PageSize: 10}
	got, err := wrapPageSQL(context.Background(), &DataSourceConfig{Dialect: "iotdb"}, finder, page)
	if err != nil {
		t.Fatalf("wrapPageSQL() error = %v", err)
	}
	if want := "SELECT temperature FROM root.sg.device1 LIMIT 10 OFFSET 20"; got != want {
		t.Fatalf("wrapPageSQL() = %q, want %q", got, want)
	}
}

func TestIoTDBParameterBinding(t *testing.T) {
	sqlstr := "SELECT temperature FROM root.sg.device1 WHERE time IN (?) AND temperature > ?"
	args := []interface{}{[]int64{10, 20}, float64(21.5)}
	gotSQL, gotValues, err := reBuildSQL(context.Background(), &DataSourceConfig{Dialect: "iotdb"}, &sqlstr, &args)
	if err != nil {
		t.Fatalf("reBuildSQL() error = %v", err)
	}
	if want := "SELECT temperature FROM root.sg.device1 WHERE time IN (?,?) AND temperature > ?"; *gotSQL != want {
		t.Fatalf("SQL = %q, want %q", *gotSQL, want)
	}
	wantValues := []interface{}{int64(10), int64(20), float64(21.5)}
	if !reflect.DeepEqual(*gotValues, wantValues) {
		t.Fatalf("values = %#v, want %#v", *gotValues, wantValues)
	}
}

func TestIoTDBInsertUsesDevicePathAndNamedColumns(t *testing.T) {
	cache := &entityStructCache{
		columns: []*fieldColumnCache{
			{columnTag: "timestamp"},
			{columnTag: "temperature"},
		},
		valuesSQL: "(?,?)",
	}
	got := wrapIoTDBInsertSQL("root.sg.device1", cache)
	if want := "INSERT INTO root.sg.device1(timestamp,temperature) VALUES(?,?)"; got != want {
		t.Fatalf("wrapIoTDBInsertSQL() = %q, want %q", got, want)
	}
}

func TestIoTDBMapInsertAlwaysIncludesColumns(t *testing.T) {
	entity := NewEntityMap("root.sg.device1")
	entity.PkColumnName = ""
	entity.Set("timestamp", int64(10))
	entity.Set("temperature", float64(21.5))

	gotSQL, gotValues, autoIncrement, err := wrapInsertEntityMapSQL(context.Background(), &DataSourceConfig{
		Dialect:           "iotdb",
		InsertSQLNoColumn: true,
	}, entity)
	if err != nil {
		t.Fatalf("wrapInsertEntityMapSQL() error = %v", err)
	}
	if want := "INSERT INTO root.sg.device1(timestamp,temperature) VALUES (?,?)"; gotSQL != want {
		t.Fatalf("SQL = %q, want %q", gotSQL, want)
	}
	if autoIncrement {
		t.Fatal("IoTDB map insert must not use auto-increment")
	}
	wantValues := []interface{}{int64(10), float64(21.5)}
	if !reflect.DeepEqual(*gotValues, wantValues) {
		t.Fatalf("values = %#v, want %#v", *gotValues, wantValues)
	}
}
