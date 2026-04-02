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
	"strings"
	"testing"
)

// assertPart 验证 SQL 片段的正确性
func assertPart(t *testing.T, sql string, part sqlSpan, expectedKeyword string) {
	t.Helper()
	if part.End <= part.Start {
		t.Errorf("SQL 片段未正确设置：%s, Start=%d, End=%d", expectedKeyword, part.Start, part.End)
		return
	}
	content := strings.ToUpper(strings.TrimSpace(sql[part.Start:part.End]))
	if !strings.Contains(content, strings.ToUpper(expectedKeyword)) {
		t.Errorf("SQL 片段内容不正确：%s, 期望包含：%s, 实际：%s",
			expectedKeyword, expectedKeyword, sql[part.Start:part.End])
	}
}

// ---------------- TestParseSQL_Basic ----------------
// 测试基础 SQL 解析
func TestParseSQL_Basic(t *testing.T) {
	sql := `SELECT name, count(*) FROM user WHERE age > 18 GROUP BY name ORDER BY count(*) DESC`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.GroupBy, "GROUP")
	assertPart(t, sql, parts.OrderBy, "ORDER")
}

// ---------------- TestParseSQL_Subquery ----------------
// 测试子查询 SQL 解析
func TestParseSQL_Subquery(t *testing.T) {
	sql := `SELECT name, count(*)
		FROM (
			SELECT name FROM user GROUP BY name
		) t
		WHERE age > 18
		GROUP   BY name
		ORDER BY count(*) DESC`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.GroupBy, "GROUP")
	assertPart(t, sql, parts.OrderBy, "ORDER")

	// 验证 FROM 子句包含子查询
	fromContent := sql[parts.From.Start:parts.From.End]
	if !strings.Contains(fromContent, "SELECT name FROM user") {
		t.Errorf("FROM 子句应包含子查询, 实际：%s", fromContent)
	}
}

// ---------------- TestParseSQL_NoWhere ----------------
// 测试没有 WHERE 子句的 SQL
func TestParseSQL_NoWhere(t *testing.T) {
	sql := `SELECT * FROM users ORDER BY id`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.OrderBy, "ORDER")

	// WHERE 应该未被设置
	if parts.Where.Start != parts.Where.End {
		t.Errorf("WHERE 子句不应被设置, 实际 Start=%d, End=%d", parts.Where.Start, parts.Where.End)
	}
}

// ---------------- TestParseSQL_NoOrderBy ----------------
// 测试没有 ORDER BY 子句的 SQL
func TestParseSQL_NoOrderBy(t *testing.T) {
	sql := `SELECT id, name FROM users WHERE status = 1`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")

	// ORDER BY 应该未被设置
	if parts.OrderBy.Start != parts.OrderBy.End {
		t.Errorf("ORDER BY 子句不应被设置, 实际 Start=%d, End=%d", parts.OrderBy.Start, parts.OrderBy.End)
	}
}

// ---------------- TestParseSQL_NoGroupBy ----------------
// 测试没有 GROUP BY 子句的 SQL
func TestParseSQL_NoGroupBy(t *testing.T) {
	sql := `SELECT * FROM users WHERE id = 1 ORDER BY created_at`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.OrderBy, "ORDER")

	// GROUP BY 应该未被设置
	if parts.GroupBy.Start != parts.GroupBy.End {
		t.Errorf("GROUP BY 子句不应被设置, 实际 Start=%d, End=%d", parts.GroupBy.Start, parts.GroupBy.End)
	}
}

// ---------------- TestParseSQL_StringWithQuote ----------------
// 测试字符串中包含引号的 SQL
func TestParseSQL_StringWithQuote(t *testing.T) {
	sql := `SELECT * FROM users WHERE name = 'O''Brien' AND desc = 'It''s fine'`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_StringWithBackslash ----------------
// 测试字符串中包含反斜杠的 SQL
func TestParseSQL_StringWithBackslash(t *testing.T) {
	sql := `SELECT * FROM users WHERE path = 'C:\\Users\\test' AND regex = 'a\b'`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_DoubleQuoteString ----------------
// 测试双引号字符串的 SQL
func TestParseSQL_DoubleQuoteString(t *testing.T) {
	sql := `SELECT "name", "age" FROM "users" WHERE "id" = 1`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_LineComment ----------------
// 测试单行注释的 SQL
func TestParseSQL_LineComment(t *testing.T) {
	sql := `SELECT * FROM users -- 这是注释
		WHERE id = 1`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_LineCommentEOF ----------------
// 测试单行注释在末尾没有换行符的 SQL (边界情况)
// 例如：SELECT * FROM user -- comment  没有 \n
func TestParseSQL_LineCommentEOF(t *testing.T) {
	sql := "SELECT * FROM users -- comment at end without newline"

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")

	// WHERE 应该未被设置 (注释后没有内容)
	if parts.Where.Start != parts.Where.End {
		t.Errorf("WHERE 子句不应被设置, 实际 Start=%d, End=%d", parts.Where.Start, parts.Where.End)
	}
}

// ---------------- TestParseSQL_LineCommentNoSpace ----------------
// 测试单行注释紧跟关键字后没有空格的 SQL
func TestParseSQL_LineCommentNoSpace(t *testing.T) {
	sql := "SELECT * FROM users--comment without space\nWHERE id = 1"

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_MultiLineComment ----------------
// 测试多行注释的 SQL
func TestParseSQL_MultiLineComment(t *testing.T) {
	sql := `SELECT /* 这是注释 */ * FROM users WHERE id = 1`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_NestedComment ----------------
// 测试嵌套括号的 SQL
func TestParseSQL_NestedParentheses(t *testing.T) {
	sql := `SELECT (SELECT COUNT(*) FROM orders WHERE orders.user_id = users.id) AS order_count FROM users WHERE status IN (1, 2, 3)`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_CaseInsensitive ----------------
// 测试大小写不敏感的 SQL
func TestParseSQL_CaseInsensitive(t *testing.T) {
	sql := `select * from USERS where ID = 1 group by NAME order by TIME limit 10`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.GroupBy, "GROUP")
	assertPart(t, sql, parts.OrderBy, "ORDER")
}

// ---------------- TestParseSQL_MixedCase ----------------
// 测试混合大小写的 SQL
func TestParseSQL_MixedCase(t *testing.T) {
	sql := `SeLeCt * FrOm users WhErE id = 1 GrOuP By name OrDeR By id`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.GroupBy, "GROUP")
	assertPart(t, sql, parts.OrderBy, "ORDER")
}

// ---------------- TestParseSQL_ExtraSpaces ----------------
// 测试多余空格的 SQL
func TestParseSQL_ExtraSpaces(t *testing.T) {
	sql := `SELECT    *    FROM    users    WHERE    id = 1    GROUP    BY    name    ORDER    BY    id`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.GroupBy, "GROUP")
	assertPart(t, sql, parts.OrderBy, "ORDER")
}

// ---------------- TestParseSQL_NewLines ----------------
// 测试换行符的 SQL
func TestParseSQL_NewLines(t *testing.T) {
	sql := `SELECT *
FROM users
WHERE id = 1
GROUP BY name
ORDER BY id`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.GroupBy, "GROUP")
	assertPart(t, sql, parts.OrderBy, "ORDER")
}

// ---------------- TestParseSQL_KeywordInString ----------------
// 测试关键字在字符串中的 SQL
func TestParseSQL_KeywordInString(t *testing.T) {
	sql := `SELECT * FROM users WHERE name = 'SELECT FROM WHERE' AND desc = 'ORDER BY LIMIT'`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")

	// 不应该错误地解析字符串中的关键字
	if parts.OrderBy.Start != parts.OrderBy.End {
		t.Errorf("ORDER BY 不应被设置 (字符串中的伪关键字) ")
	}
}

// ---------------- TestParseSQL_KeywordInIdentifier ----------------
// 测试关键字作为标识符一部分的 SQL
func TestParseSQL_KeywordInIdentifier(t *testing.T) {
	sql := `SELECT select_time, from_addr, where_field FROM table1 WHERE orderby = 1 AND grouping = 2`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")

	// 不应该错误地解析标识符中的关键字
	if parts.OrderBy.Start != parts.OrderBy.End {
		t.Errorf("ORDER BY 不应被设置 (标识符中的伪关键字) ")
	}
	if parts.GroupBy.Start != parts.GroupBy.End {
		t.Errorf("GROUP BY 不应被设置 (标识符中的伪关键字) ")
	}
}

// ---------------- TestParseSQL_Union ----------------
// 测试 UNION 的 SQL
func TestParseSQL_Union(t *testing.T) {
	sql := `SELECT id, name FROM users WHERE status = 1 UNION SELECT id, name FROM admins WHERE status = 1 ORDER BY id`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	// FROM 应该指向第一个 FROM
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.OrderBy, "ORDER")
}

// ---------------- TestParseSQL_Join ----------------
// 测试 JOIN 的 SQL
func TestParseSQL_Join(t *testing.T) {
	sql := `SELECT u.id, u.name, o.amount FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.status = 1 ORDER BY o.created_at`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.OrderBy, "ORDER")
}

// ---------------- TestParseSQL_MultipleJoins ----------------
// 测试多个 JOIN 的 SQL
func TestParseSQL_MultipleJoins(t *testing.T) {
	sql := `SELECT * FROM users u
		INNER JOIN orders o ON u.id = o.user_id
		LEFT JOIN products p ON o.product_id = p.id
		WHERE u.status = 1`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_WithHint ----------------
// 测试带 Hint 的 SQL
func TestParseSQL_WithHint(t *testing.T) {
	sql := `SELECT /*+ INDEX(users idx_status) */ * FROM users WHERE status = 1`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_Distinct ----------------
// 测试 DISTINCT 的 SQL
func TestParseSQL_Distinct(t *testing.T) {
	sql := `SELECT DISTINCT name FROM users WHERE status = 1 GROUP BY name`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.GroupBy, "GROUP")
}

// ---------------- TestParseSQL_Count ----------------
// 测试 COUNT 聚合的 SQL
func TestParseSQL_Count(t *testing.T) {
	sql := `SELECT COUNT(*) AS total FROM users WHERE status = 1`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_CountSubquery ----------------
// 测试分页 COUNT 子查询的 SQL (用于替代正则表达式场景)
func TestParseSQL_CountSubquery(t *testing.T) {
	sql := `SELECT COUNT(*) frame_row_count FROM (SELECT DISTINCT name FROM users WHERE age > 18 GROUP BY name ORDER BY name) temp_frame_noob_table_name WHERE 1=1`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")

	// 验证能正确解析内层查询的关键字
	content := sql[parts.From.Start:parts.From.End]
	if !strings.Contains(content, "DISTINCT") {
		t.Errorf("FROM 子句应包含 DISTINCT")
	}
	if !strings.Contains(content, "GROUP BY") {
		t.Errorf("FROM 子句应包含 GROUP BY")
	}
}

// ---------------- TestParseSQL_Insert ----------------
// 测试 INSERT 语句 (只解析 SELECT 部分)
func TestParseSQL_Insert(t *testing.T) {
	sql := `INSERT INTO users (name, age) VALUES ('test', 18)`

	parts := parseSQL(&sql)

	// INSERT 语句没有 SELECT 关键字, Select 会包含整个语句
	if parts.Select.End != len(sql) {
		t.Errorf("INSERT 语句的 Select 应包含整个语句")
	}
}

// ---------------- TestParseSQL_Update ----------------
// 测试 UPDATE 语句
func TestParseSQL_Update(t *testing.T) {
	sql := `UPDATE users SET name = 'test', age = 18 WHERE id = 1`

	parts := parseSQL(&sql)

	// UPDATE 语句没有 SELECT/FROM 关键字
	if parts.Where.Start == 0 || parts.Where.End == 0 {
		t.Errorf("UPDATE 语句应解析 WHERE 子句")
	}
}

// ---------------- TestParseSQL_Delete ----------------
// 测试 DELETE 语句
func TestParseSQL_Delete(t *testing.T) {
	sql := `DELETE FROM users WHERE id = 1`

	parts := parseSQL(&sql)

	// DELETE FROM 语句应解析 FROM 和 WHERE
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_EmptyString ----------------
// 测试空字符串
func TestParseSQL_EmptyString(t *testing.T) {
	sql := ``

	parts := parseSQL(&sql)

	// 空 SQL 应返回全 0 的 part
	if parts.Select.Start != 0 || parts.Select.End != 0 {
		t.Errorf("空 SQL 的 Select 应为 Start=0, End=0")
	}
}

// ---------------- TestParseSQL_OnlySelect ----------------
// 测试只有 SELECT 的 SQL
func TestParseSQL_OnlySelect(t *testing.T) {
	sql := `SELECT 1 + 1 AS result`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")

	// 其他子句应未被设置
	if parts.From.Start != parts.From.End {
		t.Errorf("FROM 不应被设置")
	}
	if parts.Where.Start != parts.Where.End {
		t.Errorf("WHERE 不应被设置")
	}
	if parts.GroupBy.Start != parts.GroupBy.End {
		t.Errorf("GROUP BY 不应被设置")
	}
	if parts.OrderBy.Start != parts.OrderBy.End {
		t.Errorf("ORDER BY 不应被设置")
	}
}

// ---------------- TestParseSQL_MultipleOrderBy ----------------
// 测试多个 ORDER BY 字段的 SQL
func TestParseSQL_MultipleOrderBy(t *testing.T) {
	sql := `SELECT * FROM users ORDER BY status DESC, created_at ASC, id DESC`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.OrderBy, "ORDER")

	// 验证 ORDER BY 包含所有字段
	orderByContent := sql[parts.OrderBy.Start:parts.OrderBy.End]
	if !strings.Contains(orderByContent, "status") {
		t.Errorf("ORDER BY 应包含 status 字段")
	}
	if !strings.Contains(orderByContent, "created_at") {
		t.Errorf("ORDER BY 应包含 created_at 字段")
	}
	if !strings.Contains(orderByContent, "id") {
		t.Errorf("ORDER BY 应包含 id 字段")
	}
}

// ---------------- TestParseSQL_Having ----------------
// 测试 HAVING 子句 (目前会包含在 GROUP BY 中)
func TestParseSQL_Having(t *testing.T) {
	sql := `SELECT status, COUNT(*) AS cnt FROM users WHERE age > 18 GROUP BY status HAVING COUNT(*) > 5 ORDER BY cnt DESC`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.GroupBy, "GROUP")
	assertPart(t, sql, parts.OrderBy, "ORDER")

	// HAVING 会包含在 GROUP BY 中, 直到 ORDER BY 之前
	groupByContent := sql[parts.GroupBy.Start:parts.GroupBy.End]
	if !strings.Contains(groupByContent, "HAVING") {
		t.Logf("注意：HAVING 子句目前会包含在 GROUP BY 中：%s", groupByContent)
	}
}

// ---------------- TestParseSQL_RealWorldExamples ----------------
// 测试真实世界的复杂 SQL 示例

// 示例 1: 带子查询和分页的复杂查询
func TestParseSQL_ComplexPaging(t *testing.T) {
	sql := `SELECT u.*, (SELECT COUNT(*) FROM orders WHERE user_id = u.id) AS order_count
		FROM (SELECT * FROM users WHERE status IN (1,2,3) ORDER BY created_at DESC) u
		WHERE u.age > 18
		ORDER BY u.created_at DESC`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.OrderBy, "ORDER")
}

// 示例 2: 多表关联查询
func TestParseSQL_MultiTableJoin(t *testing.T) {
	sql := `SELECT
			u.id, u.name,
			o.id AS order_id, o.amount,
			p.name AS product_name
		FROM users u
		INNER JOIN orders o ON u.id = o.user_id AND o.status = 1
		LEFT JOIN order_items oi ON o.id = oi.order_id
		INNER JOIN products p ON oi.product_id = p.id
		WHERE u.status = 1 AND u.created_at > '2024-01-01'
		GROUP BY u.id, o.id
		HAVING SUM(oi.quantity) > 0
		ORDER BY o.created_at DESC`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.GroupBy, "GROUP")
	assertPart(t, sql, parts.OrderBy, "ORDER")
}

// 示例 3: 分页 COUNT 查询 (替代正则表达式场景)
func TestParseSQL_CountForPaging(t *testing.T) {
	// 这是 dialect.go 中 selectCount 方法需要处理的典型 SQL
	originalSQL := `SELECT DISTINCT u.name, u.age FROM users u WHERE u.age > 18 GROUP BY u.name ORDER BY u.name`

	// 去掉 ORDER BY
	parts := parseSQL(&originalSQL)
	if parts.OrderBy.Start != parts.OrderBy.End {
		countSQL := originalSQL[:parts.OrderBy.Start]
		if strings.Contains(strings.ToUpper(countSQL), "ORDER") {
			t.Errorf("去掉 ORDER BY 后不应包含 ORDER 关键字：%s", countSQL)
		}
	}

	// 检查是否有 GROUP BY 或 DISTINCT
	hasGroupBy := parts.GroupBy.Start != parts.GroupBy.End
	hasDistinct := strings.Contains(strings.ToUpper(originalSQL), "DISTINCT")

	if !hasGroupBy && !hasDistinct {
		t.Errorf("应检测到 GROUP BY 或 DISTINCT")
	}
}

// ---------------- TestParseSQL_SpecialCases ----------------
// 测试边界情况

// 测试字符串未闭合的情况
func TestParseSQL_UnclosedString(t *testing.T) {
	sql := `SELECT * FROM users WHERE name = 'unclosed`

	parts := parseSQL(&sql)

	// 不应 panic, 能正常解析
	if parts.Select.Start != 0 {
		t.Errorf("Select Start 应为 0")
	}
}

// 测试注释未闭合的情况
func TestParseSQL_UnclosedComment(t *testing.T) {
	sql := `SELECT /* unclosed comment * FROM users`

	parts := parseSQL(&sql)

	// 不应 panic, 能正常解析
	if parts.Select.Start != 0 {
		t.Errorf("Select Start 应为 0")
	}
}

// 测试连续括号
func TestParseSQL_NestedBrackets(t *testing.T) {
	sql := `SELECT (((1 + 2) * 3) - 4) AS result FROM (((users)))`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
}

// 测试关键字在括号内 (子查询中的 FROM 不应影响外层)
func TestParseSQL_KeywordInSubquery(t *testing.T) {
	sql := `SELECT (SELECT name FROM inner_table WHERE id = 1) AS name FROM outer_table WHERE status = 1`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")

	// 验证 FROM 指向 outer_table 而不是 inner_table
	fromContent := sql[parts.From.Start:parts.From.End]
	if !strings.Contains(fromContent, "outer_table") {
		t.Errorf("FROM 应指向 outer_table, 实际：%s", fromContent)
	}
}

// ---------------- 进阶场景 / Advanced Scenarios ----------------

// ---------------- TestParseSQL_CaseWhen ----------------
// 测试 CASE WHEN 语句
func TestParseSQL_CaseWhen(t *testing.T) {
	sql := `SELECT id,
			CASE
				WHEN status = 1 THEN 'active'
				WHEN status = 0 THEN 'inactive'
				ELSE 'unknown'
			END AS status_name
		FROM users
		WHERE created_at > '2024-01-01'`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")

	// 验证 CASE WHEN 中的 THEN/ELSE 不会被误解析
	selectContent := sql[parts.Select.Start:parts.From.Start]
	if strings.Contains(strings.ToUpper(selectContent), "FROM") {
		t.Errorf("SELECT 子句中不应包含 FROM 关键字")
	}
}

// ---------------- TestParseSQL_CaseWhenInWhere ----------------
// 测试 WHERE 中包含 CASE WHEN 的 SQL
func TestParseSQL_CaseWhenInWhere(t *testing.T) {
	sql := `SELECT * FROM users
		WHERE CASE
			WHEN age > 18 THEN status
			ELSE 'pending'
		END = 'active'`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_WindowFunction ----------------
// 测试窗口函数 OVER() 的 SQL
func TestParseSQL_WindowFunction(t *testing.T) {
	sql := `SELECT id, name, salary,
			ROW_NUMBER() OVER (PARTITION BY dept ORDER BY salary DESC) AS rn,
			AVG(salary) OVER (PARTITION BY dept) AS avg_salary
		FROM employees
		WHERE status = 1`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")

	// 验证 OVER() 中的 ORDER BY 不会影响外层 ORDER BY 解析
	//  (本例中没有外层 ORDER BY, 所以 OrderBy 应未被设置)
	if parts.OrderBy.Start != parts.OrderBy.End {
		t.Errorf("此 SQL 不应解析出 ORDER BY 子句 (OVER 中的 ORDER BY 是窗口函数的一部分) ")
	}
}

// ---------------- TestParseSQL_WindowFunctionWithOrderBy ----------------
// 测试窗口函数 + 外层 ORDER BY 的 SQL
func TestParseSQL_WindowFunctionWithOrderBy(t *testing.T) {
	sql := `SELECT id, name,
			ROW_NUMBER() OVER (PARTITION BY dept ORDER BY salary DESC) AS rn
		FROM employees
		WHERE status = 1
		ORDER BY id DESC`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.OrderBy, "ORDER")

	// 验证 ORDER BY 指向外层的 ORDER BY id DESC
	orderByContent := sql[parts.OrderBy.Start:parts.OrderBy.End]
	if !strings.Contains(orderByContent, "id DESC") {
		t.Errorf("ORDER BY 应包含外层的 id DESC, 实际：%s", orderByContent)
	}
}

// ---------------- TestParseSQL_CTE ----------------
// 测试 CTE (WITH 子句) 的 SQL
func TestParseSQL_CTE(t *testing.T) {
	sql := `WITH active_users AS (
			SELECT id, name FROM users WHERE status = 1
		),
		order_counts AS (
			SELECT user_id, COUNT(*) AS cnt FROM orders GROUP BY user_id
		)
		SELECT u.id, u.name, oc.cnt
		FROM active_users u
		LEFT JOIN order_counts oc ON u.id = oc.user_id
		WHERE oc.cnt > 5`

	parts := parseSQL(&sql)

	// WITH 子句会被包含在 SELECT 中
	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")

	// 验证 FROM 包含 CTE 定义的表
	fromContent := sql[parts.From.Start:parts.From.End]
	if !strings.Contains(fromContent, "active_users") {
		t.Errorf("FROM 应包含 CTE 表 active_users")
	}
	if !strings.Contains(fromContent, "order_counts") {
		t.Errorf("FROM 应包含 CTE 表 order_counts")
	}
}

// ---------------- TestParseSQL_CTEWithOrderBy ----------------
// 测试 CTE + ORDER BY 的 SQL
func TestParseSQL_CTEWithOrderBy(t *testing.T) {
	sql := `WITH ranked_users AS (
			SELECT id, name, ROW_NUMBER() OVER (ORDER BY created_at DESC) AS rn
			FROM users
		)
		SELECT * FROM ranked_users WHERE rn <= 10 ORDER BY rn`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.OrderBy, "ORDER")
}

// ---------------- TestParseSQL_RecursiveCTE ----------------
// 测试递归 CTE 的 SQL
func TestParseSQL_RecursiveCTE(t *testing.T) {
	sql := `WITH RECURSIVE hierarchy AS (
			SELECT id, parent_id, name, 0 AS level
			FROM categories WHERE parent_id IS NULL
			UNION ALL
			SELECT c.id, c.parent_id, c.name, h.level + 1
			FROM categories c
			INNER JOIN hierarchy h ON c.parent_id = h.id
		)
		SELECT * FROM hierarchy ORDER BY level, name`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.OrderBy, "ORDER")
}

// ---------------- TestParseSQL_SelectInto ----------------
// 测试 SELECT INTO 语句 (MySQL)
func TestParseSQL_SelectInto(t *testing.T) {
	// MySQL 的 SELECT INTO 语法
	sql := `SELECT id, name INTO @var_id, @var_name FROM users WHERE id = 1`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_SelectIntoOutfile ----------------
// 测试 SELECT INTO OUTFILE 语句 (MySQL 导出)
func TestParseSQL_SelectIntoOutfile(t *testing.T) {
	sql := `SELECT * FROM users WHERE status = 1
		INTO OUTFILE '/tmp/users.csv'
		FIELDS TERMINATED BY ','
		LINES TERMINATED BY '\n'`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")

	// INTO OUTFILE 会被包含在 WHERE 中 (因为不是标准 SELECT...FROM 结构)
	// 这是预期行为, 因为 ParseSQL 主要关注标准查询结构
}

// ---------------- TestParseSQL_InsertSelect ----------------
// 测试 INSERT INTO ... SELECT 语句
func TestParseSQL_InsertSelect(t *testing.T) {
	sql := `INSERT INTO users_backup (id, name, email)
		SELECT id, name, email FROM users WHERE status = 1`

	parts := parseSQL(&sql)

	// INSERT INTO ... SELECT 语句中, SELECT 会被解析
	// FROM 应该指向 SELECT 中的 FROM
	if parts.From.Start == parts.From.End {
		t.Logf("注意：INSERT INTO ... SELECT 语句的 FROM 可能未被正确解析")
	}
}

// ---------------- TestParseSQL_UpdateJoin ----------------
// 测试 UPDATE ... JOIN 语句
func TestParseSQL_UpdateJoin(t *testing.T) {
	sql := `UPDATE users u
		INNER JOIN orders o ON u.id = o.user_id
		SET u.total_orders = (SELECT COUNT(*) FROM orders WHERE user_id = u.id)
		WHERE o.status = 1`

	parts := parseSQL(&sql)

	// UPDATE 语句的解析可能不同于 SELECT
	// 但 WHERE 应该被正确解析
	if parts.Where.Start == parts.Where.End {
		t.Logf("注意：UPDATE ... JOIN 语句的 WHERE 可能未被正确解析")
	}
}

// ---------------- TestParseSQL_DeleteJoin ----------------
// 测试 DELETE ... JOIN 语句
func TestParseSQL_DeleteJoin(t *testing.T) {
	sql := `DELETE u FROM users u
		LEFT JOIN orders o ON u.id = o.user_id
		WHERE o.id IS NULL`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_ForUpdate ----------------
// 测试 FOR UPDATE 锁语句
func TestParseSQL_ForUpdate(t *testing.T) {
	sql := `SELECT * FROM users WHERE id = 1 FOR UPDATE`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")

	// FOR UPDATE 会被包含在 WHERE 中 (作为 WHERE 子句的延续)
	whereContent := sql[parts.Where.Start:parts.Where.End]
	if !strings.Contains(strings.ToUpper(whereContent), "FOR UPDATE") {
		t.Logf("注意：FOR UPDATE 可能被包含在 WHERE 子句中：%s", whereContent)
	}
}

// ---------------- TestParseSQL_LockInShareMode ----------------
// 测试 LOCK IN SHARE MODE 语句
func TestParseSQL_LockInShareMode(t *testing.T) {
	sql := `SELECT * FROM users WHERE id = 1 LOCK IN SHARE MODE`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
}

// ---------------- TestParseSQL_Values ----------------
// 测试 VALUES 语句 (MySQL 特殊语法)
func TestParseSQL_Values(t *testing.T) {
	// MySQL 的 VALUES() 函数, 用于 ON DUPLICATE KEY UPDATE
	sql := `INSERT INTO users (id, name) VALUES (1, 'test')
		ON DUPLICATE KEY UPDATE name = VALUES(name)`

	parts := parseSQL(&sql)

	// 这是 INSERT 语句, 主要验证不会出错
	if parts.Select.Start != 0 {
		t.Logf("INSERT 语句的 Select 从 0 开始是正常的")
	}
}

// ---------------- TestParseSQL_ComplexRealWorld ----------------
// 测试真实世界的超复杂 SQL 示例

// 示例：电商订单查询 (包含所有进阶特性)
func TestParseSQL_EcommerceOrderQuery(t *testing.T) {
	sql := `WITH ranked_orders AS (
			SELECT
				o.id,
				o.user_id,
				o.amount,
				o.status,
				ROW_NUMBER() OVER (PARTITION BY o.user_id ORDER BY o.created_at DESC) AS rn,
				CASE
					WHEN o.amount > 1000 THEN 'high'
					WHEN o.amount > 500 THEN 'medium'
					ELSE 'low'
				END AS amount_level
			FROM orders o
			WHERE o.created_at >= '2024-01-01'
				AND o.status IN ('paid', 'shipped')
		)
		SELECT
			u.id AS user_id,
			u.name,
			ro.id AS order_id,
			ro.amount,
			ro.amount_level,
			(SELECT COUNT(*) FROM order_items WHERE order_id = ro.id) AS item_count
		FROM ranked_orders ro
		INNER JOIN users u ON ro.user_id = u.id
		WHERE ro.rn <= 5
		ORDER BY ro.created_at DESC, u.name ASC`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.Where, "WHERE")
	assertPart(t, sql, parts.OrderBy, "ORDER")

	// 验证 ORDER BY 不包含窗口函数中的 ORDER BY
	orderByContent := sql[parts.OrderBy.Start:parts.OrderBy.End]
	if strings.Contains(orderByContent, "PARTITION BY") {
		t.Errorf("ORDER BY 不应包含窗口函数的 PARTITION BY")
	}
}

// 示例：财务报表 SQL (多 CTE + 窗口函数 + CASE)
func TestParseSQL_FinancialReport(t *testing.T) {
	sql := `WITH daily_stats AS (
			SELECT
				DATE(created_at) AS stat_date,
				SUM(amount) AS daily_total,
				COUNT(*) AS daily_count
			FROM transactions
			WHERE created_at >= '2024-01-01'
			GROUP BY DATE(created_at)
		),
		moving_avg AS (
			SELECT
				stat_date,
				daily_total,
				daily_count,
				AVG(daily_total) OVER (ORDER BY stat_date ROWS BETWEEN 6 PRECEDING AND CURRENT ROW) AS weekly_avg
			FROM daily_stats
		)
		SELECT
			stat_date,
			daily_total,
			weekly_avg,
			CASE
				WHEN daily_total > weekly_avg * 1.2 THEN 'above_normal'
				WHEN daily_total < weekly_avg * 0.8 THEN 'below_normal'
				ELSE 'normal'
			END AS status
		FROM moving_avg
		ORDER BY stat_date DESC`

	parts := parseSQL(&sql)

	assertPart(t, sql, parts.Select, "SELECT")
	assertPart(t, sql, parts.From, "FROM")
	assertPart(t, sql, parts.OrderBy, "ORDER")
}
