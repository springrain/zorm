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
	"errors"
	"strconv"

	"gitee.com/chunanyong/zorm/decimal"
)

// FuncDecimalValue 设置decimal类型接收值,复写函数自定义decimal实现,例如github.com/shopspring/decimal,返回的是指针
var FuncDecimalValue = func(ctx context.Context, dialect string) interface{} {
	return &decimal.Decimal{}
}

// OverrideFunc 重写ZORM的函数,用于风险监控,只要查看这个函数的调用,就知道哪些地方重写了函数,避免项目混乱.当你使用这个函数时,你必须知道自己在做什么
// funcName 是需要重写的方法命,funcObject是对应的函数. 返回值bool是否重写成功,interface{}是重写前的函数
// 一般是在init里调用重写
func OverrideFunc(funcName string, funcObject interface{}) (bool, interface{}, error) {
	if funcName == "" {
		return false, nil, errors.New("->OverrideFunc-->funcName不能为空")
	}

	// oldFunc 老的函数
	var oldFunc interface{} = nil
	switch funcName {
	case "Transaction":
		newFunc, ok := funcObject.(func(ctx context.Context, doTransaction func(ctx context.Context) (interface{}, error)) (interface{}, error))
		if ok {
			oldFunc = transaction
			transaction = newFunc
		}
	case "QueryRow":
		newFunc, ok := funcObject.(func(ctx context.Context, finder *Finder, entity interface{}) (bool, error))
		if ok {
			oldFunc = queryRow
			queryRow = newFunc
		}
	case "Query":
		newFunc, ok := funcObject.(func(ctx context.Context, finder *Finder, rowsSlicePtr interface{}, page *Page) error)
		if ok {
			oldFunc = query
			query = newFunc
		}

	case "QueryRowMap":
		newFunc, ok := funcObject.(func(ctx context.Context, finder *Finder) (map[string]interface{}, error))
		if ok {
			oldFunc = queryRowMap
			queryRowMap = newFunc
		}
	case "QueryMap":
		newFunc, ok := funcObject.(func(ctx context.Context, finder *Finder, page *Page) ([]map[string]interface{}, error))
		if ok {
			oldFunc = queryMap
			queryMap = newFunc
		}
	case "UpdateFinder":
		newFunc, ok := funcObject.(func(ctx context.Context, finder *Finder) (int, error))
		if ok {
			oldFunc = updateFinder
			updateFinder = newFunc
		}
	case "Insert":
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityStruct) (int, error))
		if ok {
			oldFunc = insert
			insert = newFunc
		}
	case "InsertSlice":
		newFunc, ok := funcObject.(func(ctx context.Context, entityStructSlice []IEntityStruct) (int, error))
		if ok {
			oldFunc = insertSlice
			insertSlice = newFunc
		}
	case "Update":
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityStruct) (int, error))
		if ok {
			oldFunc = update
			update = newFunc
		}
	case "UpdateNotZeroValue":
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityStruct) (int, error))
		if ok {
			oldFunc = updateNotZeroValue
			updateNotZeroValue = newFunc
		}
	case "Delete":
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityStruct) (int, error))
		if ok {
			oldFunc = delete
			delete = newFunc
		}

	case "InsertEntityMap":
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityMap) (int, error))
		if ok {
			oldFunc = insertEntityMap
			insertEntityMap = newFunc
		}
	case "InsertEntityMapSlice":
		newFunc, ok := funcObject.(func(ctx context.Context, entity []IEntityMap) (int, error))
		if ok {
			oldFunc = insertEntityMapSlice
			insertEntityMapSlice = newFunc
		}
	case "UpdateEntityMap":
		newFunc, ok := funcObject.(func(ctx context.Context, entity IEntityMap) (int, error))
		if ok {
			oldFunc = updateEntityMap
			updateEntityMap = newFunc
		}
	default:
		return false, oldFunc, errors.New("->OverrideFunc-->函数" + funcName + "暂不支持重写或不存在")
	}
	if oldFunc == nil {
		return false, oldFunc, errors.New("->OverrideFunc-->请检查传入的" + funcName + "函数实现,断言转换失败.")
	}
	return true, oldFunc, nil
}

// typeConvertInt64toInt int64 转 int
func typeConvertInt64toInt(from int64) (int, error) {
	strInt64 := strconv.FormatInt(from, 10)
	return strconv.Atoi(strInt64)
}

/*
func typeConvertFloat32(i interface{}) (float32, error) {
	if i == nil {
		return 0, nil
	}
	if v, ok := i.(float32); ok {
		return v, nil
	}
	v, err := typeConvertString(i)
	if err != nil {
		return 0, err
	}
	vf, err := strconv.ParseFloat(v, 32)
	return float32(vf), err
}

func typeConvertFloat64(i interface{}) (float64, error) {
	if i == nil {
		return 0, nil
	}
	if v, ok := i.(float64); ok {
		return v, nil
	}
	v, err := typeConvertString(i)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(v, 64)
}

func typeConvertDecimal(i interface{}) (decimal.Decimal, error) {
	if i == nil {
		return decimal.Zero, nil
	}
	if v, ok := i.(decimal.Decimal); ok {
		return v, nil
	}
	v, err := typeConvertString(i)
	if err != nil {
		return decimal.Zero, err
	}
	return decimal.NewFromString(v)
}

func typeConvertInt64(i interface{}) (int64, error) {
	if i == nil {
		return 0, nil
	}
	if v, ok := i.(int64); ok {
		return v, nil
	}
	v, err := typeConvertInt(i)
	if err != nil {
		return 0, err
	}
	return int64(v), err
}

func typeConvertString(i interface{}) (string, error) {
	if i == nil {
		return "", nil
	}
	switch value := i.(type) {
	case int:
		return strconv.Itoa(value), nil
	case int8:
		return strconv.Itoa(int(value)), nil
	case int16:
		return strconv.Itoa(int(value)), nil
	case int32:
		return strconv.Itoa(int(value)), nil
	case int64:
		return strconv.Itoa(int(value)), nil
	case uint:
		return strconv.FormatUint(uint64(value), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(value), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(value), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(value), 10), nil
	case uint64:
		return strconv.FormatUint(uint64(value), 10), nil
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(value), nil
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	default:
		return fmt.Sprintf("%v", value), nil
	}
}

//false: "", 0, false, off
func typeConvertBool(i interface{}) (bool, error) {
	if i == nil {
		return false, nil
	}
	if v, ok := i.(bool); ok {
		return v, nil
	}
	s, err := typeConvertString(i)
	if err != nil {
		return false, err
	}
	if s != "" && s != "0" && s != "false" && s != "off" {
		return true, err
	}
	return false, err
}

func typeConvertInt(i interface{}) (int, error) {
	if i == nil {
		return 0, nil
	}
	switch value := i.(type) {
	case int:
		return value, nil
	case int8:
		return int(value), nil
	case int16:
		return int(value), nil
	case int32:
		return int(value), nil
	case int64:
		return int(value), nil
	case uint:
		return int(value), nil
	case uint8:
		return int(value), nil
	case uint16:
		return int(value), nil
	case uint32:
		return int(value), nil
	case uint64:
		return int(value), nil
	case float32:
		return int(value), nil
	case float64:
		return int(value), nil
	case bool:
		if value {
			return 1, nil
		}
		return 0, nil
	default:
		v, err := typeConvertString(value)
		if err != nil {
			return 0, err
		}
		return strconv.Atoi(v)
	}
}



func typeConvertTime(i interface{}, format string, TZLocation ...*time.Location) (time.Time, error) {
	s, err := typeConvertString(i)
	if err != nil {
		return time.Time{}, err
	}
	return typeConvertStrToTime(s, format, TZLocation...)
}

func typeConvertStrToTime(str string, format string, TZLocation ...*time.Location) (time.Time, error) {
	if len(TZLocation) > 0 {
		return time.ParseInLocation(format, str, TZLocation[0])
	}
	return time.ParseInLocation(format, str, time.Local)
}

func encodeString(s string) []byte {
	return []byte(s)
}

func decodeToString(b []byte) string {
	return string(b)
}

func encodeBool(b bool) []byte {
	if b {
		return []byte{1}
	}
	return []byte{0}

}

func encodeInt(i int) []byte {
	if i <= math.MaxInt8 {
		return encodeInt8(int8(i))
	} else if i <= math.MaxInt16 {
		return encodeInt16(int16(i))
	} else if i <= math.MaxInt32 {
		return encodeInt32(int32(i))
	} else {
		return encodeInt64(int64(i))
	}
}

func encodeUint(i uint) []byte {
	if i <= math.MaxUint8 {
		return encodeUint8(uint8(i))
	} else if i <= math.MaxUint16 {
		return encodeUint16(uint16(i))
	} else if i <= math.MaxUint32 {
		return encodeUint32(uint32(i))
	} else {
		return encodeUint64(uint64(i))
	}
}

func encodeInt8(i int8) []byte {
	return []byte{byte(i)}
}

func encodeUint8(i uint8) []byte {
	return []byte{byte(i)}
}

func encodeInt16(i int16) []byte {
	bytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(bytes, uint16(i))
	return bytes
}

func encodeUint16(i uint16) []byte {
	bytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(bytes, i)
	return bytes
}

func encodeInt32(i int32) []byte {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, uint32(i))
	return bytes
}

func encodeUint32(i uint32) []byte {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, i)
	return bytes
}

func encodeInt64(i int64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, uint64(i))
	return bytes
}

func encodeUint64(i uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, i)
	return bytes
}

func encodeFloat32(f float32) []byte {
	bits := math.Float32bits(f)
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, bits)
	return bytes
}

func encodeFloat64(f float64) []byte {
	bits := math.Float64bits(f)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)
	return bytes
}

func encode(vs ...interface{}) []byte {
	buf := new(bytes.Buffer)
	for i := 0; i < len(vs); i++ {
		switch value := vs[i].(type) {
		case int:
			buf.Write(encodeInt(value))
		case int8:
			buf.Write(encodeInt8(value))
		case int16:
			buf.Write(encodeInt16(value))
		case int32:
			buf.Write(encodeInt32(value))
		case int64:
			buf.Write(encodeInt64(value))
		case uint:
			buf.Write(encodeUint(value))
		case uint8:
			buf.Write(encodeUint8(value))
		case uint16:
			buf.Write(encodeUint16(value))
		case uint32:
			buf.Write(encodeUint32(value))
		case uint64:
			buf.Write(encodeUint64(value))
		case bool:
			buf.Write(encodeBool(value))
		case string:
			buf.Write(encodeString(value))
		case []byte:
			buf.Write(value)
		case float32:
			buf.Write(encodeFloat32(value))
		case float64:
			buf.Write(encodeFloat64(value))
		default:
			if err := binary.Write(buf, binary.LittleEndian, value); err != nil {
				buf.Write(encodeString(fmt.Sprintf("%v", value)))
			}
		}
	}
	return buf.Bytes()
}

func isNumeric(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < byte('0') || s[i] > byte('9') {
			return false
		}
	}
	return true
}
func typeConvertTimeDuration(i interface{}) time.Duration {
	return time.Duration(typeConvertInt64(i))
}

func typeConvertBytes(i interface{}) []byte {
	if i == nil {
		return nil
	}
	if r, ok := i.([]byte); ok {
		return r
	}
	return encode(i)

}

func typeConvertStrings(i interface{}) []string {
	if i == nil {
		return nil
	}
	if r, ok := i.([]string); ok {
		return r
	} else if r, ok := i.([]interface{}); ok {
		strs := make([]string, len(r))
		for k, v := range r {
			strs[k] = typeConvertString(v)
		}
		return strs
	}
	return []string{fmt.Sprintf("%v", i)}
}

func typeConvertInt8(i interface{}) int8 {
	if i == nil {
		return 0
	}
	if v, ok := i.(int8); ok {
		return v
	}
	return int8(typeConvertInt(i))
}

func typeConvertInt16(i interface{}) int16 {
	if i == nil {
		return 0
	}
	if v, ok := i.(int16); ok {
		return v
	}
	return int16(typeConvertInt(i))
}

func typeConvertInt32(i interface{}) int32 {
	if i == nil {
		return 0
	}
	if v, ok := i.(int32); ok {
		return v
	}
	return int32(typeConvertInt(i))
}

func typeConvertUint(i interface{}) uint {
	if i == nil {
		return 0
	}
	switch value := i.(type) {
	case int:
		return uint(value)
	case int8:
		return uint(value)
	case int16:
		return uint(value)
	case int32:
		return uint(value)
	case int64:
		return uint(value)
	case uint:
		return value
	case uint8:
		return uint(value)
	case uint16:
		return uint(value)
	case uint32:
		return uint(value)
	case uint64:
		return uint(value)
	case float32:
		return uint(value)
	case float64:
		return uint(value)
	case bool:
		if value {
			return 1
		}
		return 0
	default:
		v, _ := strconv.ParseUint(typeConvertString(value), 10, 64)
		return uint(v)
	}
}

func typeConvertUint8(i interface{}) uint8 {
	if i == nil {
		return 0
	}
	if v, ok := i.(uint8); ok {
		return v
	}
	return uint8(typeConvertUint(i))
}

func typeConvertUint16(i interface{}) uint16 {
	if i == nil {
		return 0
	}
	if v, ok := i.(uint16); ok {
		return v
	}
	return uint16(typeConvertUint(i))
}

func typeConvertUint32(i interface{}) uint32 {
	if i == nil {
		return 0
	}
	if v, ok := i.(uint32); ok {
		return v
	}
	return uint32(typeConvertUint(i))
}

func typeConvertUint64(i interface{}) uint64 {
	if i == nil {
		return 0
	}
	if v, ok := i.(uint64); ok {
		return v
	}
	return uint64(typeConvertUint(i))
}
*/
