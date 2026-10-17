package zorm

import (
	"fmt"
	"context"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// P0-3: wrapInsertEntityMapSliceSQL 预分配 values 总容量
// P0-4: wrapParamSQL 用 strconv.AppendInt 替代 strconv.Itoa
//
// 这两个基准都直接调用生产函数,不复制实现.
// ---------------------------------------------------------------------------

// p0BuildEntityMaps 构造 n 行、每行 fields 个字段的 EntityMap 切片
func p0BuildEntityMaps(tableName string, rows, fields int) []IEntityMap {
	out := make([]IEntityMap, rows)
	for r := 0; r < rows; r++ {
		em := NewEntityMap(tableName)
		for f := 0; f < fields; f++ {
			em.Set("col"+strconv.Itoa(f), r*1000+f)
		}
		out[r] = em
	}
	return out
}

// --- P0-3 正确性: 预分配后 values 的内容与顺序必须与原来完全一致 ---

func Test_P0_3_ValuesContentAndOrder(t *testing.T) {
	const rows, fields = 50, 6
	maps := p0BuildEntityMaps("t_user", rows, fields)

	_, valuesPtr, err := wrapInsertEntityMapSliceSQL(context.Background(), &DataSourceConfig{}, maps)
	if err != nil {
		t.Fatal(err)
	}
	values := *valuesPtr

	if len(values) != rows*fields {
		t.Fatalf("values 长度 = %d, 期望 %d", len(values), rows*fields)
	}
	// 逐值校验: 第 r 行第 f 列应是 r*1000+f
	for r := 0; r < rows; r++ {
		for f := 0; f < fields; f++ {
			got := values[r*fields+f]
			want := r*1000 + f
			if got != want {
				t.Fatalf("values[%d] = %v, 期望 %v (行%d 列%d)", r*fields+f, got, want, r, f)
			}
		}
	}
}

// 容量断言: 预分配后容量应恰好为 rows*fields (精确上界, 无过度分配)
func Test_P0_3_CapacityExact(t *testing.T) {
	const rows, fields = 50, 6
	maps := p0BuildEntityMaps("t_user", rows, fields)
	_, valuesPtr, err := wrapInsertEntityMapSliceSQL(context.Background(), &DataSourceConfig{}, maps)
	if err != nil {
		t.Fatal(err)
	}
	if c := cap(*valuesPtr); c != rows*fields {
		t.Fatalf("cap = %d, 期望精确的 %d (无过度分配)", c, rows*fields)
	}
}

// 边界: 单行时不应触发任何重分配 (cap 已由 wrapInsertValueEntityMapSQL 给足)
func Test_P0_3_SingleRow(t *testing.T) {
	const fields = 6
	maps := p0BuildEntityMaps("t_user", 1, fields)
	_, valuesPtr, err := wrapInsertEntityMapSliceSQL(context.Background(), &DataSourceConfig{}, maps)
	if err != nil {
		t.Fatal(err)
	}
	if len(*valuesPtr) != fields {
		t.Fatalf("单行 values 长度 = %d, 期望 %d", len(*valuesPtr), fields)
	}
}

// 空切片必须仍然返回错误, 行为不变
func Test_P0_3_EmptyStillErrors(t *testing.T) {
	_, _, err := wrapInsertEntityMapSliceSQL(context.Background(), &DataSourceConfig{}, nil)
	if err == nil {
		t.Fatal("空切片应当返回错误, 实际返回 nil")
	}
}

// 生成的 SQL 必须仍然正确 (逗号分隔的 VALUES 组数 = 行数)
func Test_P0_3_SQLShape(t *testing.T) {
	const rows, fields = 20, 4
	maps := p0BuildEntityMaps("t_user", rows, fields)
	sqlPtr, _, err := wrapInsertEntityMapSliceSQL(context.Background(), &DataSourceConfig{}, maps)
	if err != nil {
		t.Fatal(err)
	}
	sql := *sqlPtr
	if !strings.HasPrefix(sql, "INSERT INTO t_user") {
		t.Fatalf("SQL 前缀错误: %.60s", sql)
	}
	// 每行一组 "(?,?,?,?)", 组间用逗号分隔
	group := "(" + strings.Repeat("?,", fields-1) + "?)"
	if n := strings.Count(sql, group); n != rows {
		t.Fatalf("VALUES 组数 = %d, 期望 %d. SQL=%.120s", n, rows, sql)
	}
}

// --- P0-4 正确性: AppendInt 产出的占位符下标必须与 Itoa 完全一致 ---

// 用旧的 Itoa 实现作为参照, 穷举对比下标边界
func itoaReference(symbols string, valueLen int, sqlParamIndex int) string {
	var b strings.Builder
	if valueLen == 1 {
		b.WriteString(symbols)
		b.WriteString(strconv.Itoa(sqlParamIndex))
	} else if valueLen > 1 {
		for j := 0; j < valueLen; j++ {
			if j == 0 {
				b.WriteString(symbols)
			} else {
				b.WriteByte(',')
				b.WriteString(symbols)
			}
			b.WriteString(strconv.Itoa(sqlParamIndex + j))
		}
	}
	return b.String()
}

func Test_P0_4_AppendIntMatchesItoa(t *testing.T) {
	symbolsSet := []string{"?", "$", "@p", ":", "'?'"}
	// 覆盖到 int64 十进制的位数边界: 1位/2位/3位/10位/18位/19位
	indices := []int{0, 1, 9, 10, 99, 100, 999, 1000, 1 << 20, 999999999, 1 << 40, 1<<62 + 7, 1<<63 - 1}
	lengths := []int{1, 2, 3, 8}

	for _, symbols := range symbolsSet {
		for _, idx := range indices {
			for _, L := range lengths {
				var b strings.Builder
				p := idx
				var vals []interface{}
				// 构造一个长度为 L 的可索引反射值
				slice := make([]int, L)
				for i := range slice {
					slice[i] = i
				}
				v := reflect.ValueOf(slice)
				wrapParamSQL(symbols, L, &p, &b, &v, &vals, true, false)
				got := b.String()
				want := itoaReference(symbols, L, idx)
				if got != want {
					t.Fatalf("symbols=%q idx=%d L=%d\n got=%q\nwant=%q", symbols, idx, L, got, want)
				}
				if p != idx+L {
					t.Fatalf("下标推进错误: %d, 期望 %d", p, idx+L)
				}
			}
		}
	}
}

// hasParamIndex=false 时不写下标, 仅写占位符 (行为不变)
func Test_P0_4_NoParamIndex(t *testing.T) {
	var b strings.Builder
	p := 5
	var vals []interface{}
	slice := []int{1, 2, 3}
	v := reflect.ValueOf(slice)
	wrapParamSQL("?", 3, &p, &b, &v, &vals, false, false)
	if got, want := b.String(), "?,?,?"; got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
	if len(vals) != 3 {
		t.Fatalf("newValues 长度 = %d, 期望 3", len(vals))
	}
}

// valueLen==1 时只写一个占位符 (行为不变)
func Test_P0_4_SingleValueLen(t *testing.T) {
	var b strings.Builder
	p := 7
	var vals []interface{}
	slice := []int{1}
	v := reflect.ValueOf(slice)
	wrapParamSQL("$", 1, &p, &b, &v, &vals, true, false)
	if got, want := b.String(), "$7"; got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
	if p != 8 {
		t.Fatalf("下标 = %d, 期望 8", p)
	}
}

// --- 基准 ---

func BenchmarkP0_3_InsertEntityMapSlice(b *testing.B) {
	ctx := context.Background()
	const rows, fields = 200, 8
	maps := p0BuildEntityMaps("t_user", rows, fields)
	cfg := &DataSourceConfig{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := wrapInsertEntityMapSliceSQL(ctx, cfg, maps); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkP0_4_WrapParamSQL(b *testing.B) {
	const args = 300
	slice := make([]int, 1)
	v := reflect.ValueOf(slice)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		sb.Grow(stringBuilderGrowLen)
		p := 1
		var vals []interface{}
		for a := 0; a < args; a++ {
			wrapParamSQL("$", 1, &p, &sb, &v, &vals, true, false)
		}
		_ = sb.String()
	}
}

// 并发验证: wrapParamSQL 的 indexBuf 是函数局部变量, 必须在并发下无竞争.
// 真正兜底的是下面的"下标连续性"断言: 若 indexBuf 被误提到包级, 并发写会污染
// 已产出的占位符数字, 该断言会失败(实测: 提到包级后报 "下标不连续: 31 后面是 2").
// 注意 -race 对这种"写入 Builder 前先覆写共享缓冲"的短模式可能漏报, 不能只靠它.
// 本测试同时配合 -race 运行.
func Test_P0_4_ConcurrentNoRace(t *testing.T) {
	const goroutines = 32
	const perG = 200
	done := make(chan string, goroutines)
	for g := 0; g < goroutines; g++ {
		go func(base int) {
			slice := make([]int, base%4+1)
			for i := range slice {
				slice[i] = i
			}
			v := reflect.ValueOf(slice)
			for k := 0; k < perG; k++ {
				var b strings.Builder
				p := base
				var vals []interface{}
				wrapParamSQL("$", len(slice), &p, &b, &v, &vals, true, false)
				done <- b.String()
			}
		}(g)
	}
	// 校验每个结果都自洽: $n,$n+1,...
	for i := 0; i < goroutines*perG; i++ {
		s := <-done
		if !strings.HasPrefix(s, "$") {
			t.Fatalf("结果格式错误: %q", s)
		}
		parts := strings.Split(s, ",")
		var first int
		if _, err := fmt.Sscanf(parts[0], "$%d", &first); err != nil {
			t.Fatalf("解析失败 %q: %v", parts[0], err)
		}
		for j, part := range parts[1:] {
			var got int
			if _, err := fmt.Sscanf(part, "$%d", &got); err != nil {
				t.Fatalf("解析失败 %q: %v", part, err)
			}
			if got != first+j+1 {
				t.Fatalf("下标不连续: %d 后面是 %d", first+j, got)
			}
		}
	}
}
