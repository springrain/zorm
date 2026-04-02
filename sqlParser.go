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

// sqlSpan 表示 SQL 中某个片段的位置范围 (左闭右开)
// sqlSpan represents the position range of a SQL fragment (left-closed, right-open)
type sqlSpan struct {
	Start int // 起始位置 (包含) / Start position (inclusive)
	End   int // 结束位置 (不包含) / End position (exclusive)
}

// sqlPart 表示 SQL 语句的各个子句片段
// sqlPart represents the fragments of each clause in a SQL statement
type sqlPart struct {
	Select  sqlSpan // SELECT 子句 / SELECT clause
	From    sqlSpan // FROM 子句 / FROM clause
	Where   sqlSpan // WHERE 子句 / WHERE clause
	GroupBy sqlSpan // GROUP BY 子句 / GROUP BY clause
	OrderBy sqlSpan // ORDER BY 子句 / ORDER BY clause
}

// sqlScanner SQL 词法扫描器, 用于逐个字符解析 SQL
// sqlScanner SQL lexical scanner for parsing SQL character by character
type sqlScanner struct {
	s     *string // 原始 SQL 字符串指针 / Original SQL string
	i     int     // 当前扫描位置 / Current scan position
	n     int     // SQL 字符串总长度 / Total length of SQL string
	depth int     // 括号嵌套深度, 用于处理子查询 / Parentheses nesting depth for handling subqueries
}

// ================= 基础能力 / Basic Capabilities =================

// isIdentChar 判断字符是否为标识符字符 (字母、数字、下划线)
// isIdentChar checks if a character is an identifier character (letter, digit, underscore)
func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

// skipString 跳过字符串字面量 (支持单引号 ' 和双引号 ")
// 处理转义: \' 和 "" (SQL 标准双单引号转义)
// skipString skips string literals (supports single quote ' and double quote ")
// Handles escapes: \' and "" (SQL standard double single-quote escape)
func (sc *sqlScanner) skipString() {
	quote := (*sc.s)[sc.i] // 记录字符串的引号类型 / Record the quote type of the string
	sc.i++                 // 跳过开引号 / Skip opening quote

	for sc.i < sc.n {
		// \ 转义: 处理 \' 这种情况
		// Backslash escape: handles cases like \'
		if (*sc.s)[sc.i] == '\\' && sc.i+1 < sc.n {
			sc.i += 2 // 跳过转义字符和下一个字符 / Skip escape character and next character
			continue
		}

		// '' 转义 (SQL 标准) : 处理 'O''Brien' 这种情况
		// '' escape (SQL standard): handles cases like 'O''Brien'
		if (*sc.s)[sc.i] == quote {
			if sc.i+1 < sc.n && (*sc.s)[sc.i+1] == quote {
				sc.i += 2 // 跳过两个连续的引号 / Skip two consecutive quotes
				continue
			}
			sc.i++ // 跳过闭引号 / Skip closing quote
			return // 字符串结束 / End of string
		}

		sc.i++ // 继续扫描下一个字符 / Continue to scan next character
	}
	// 字符串未闭合也会正常退出, 不会报错 / Exits normally even if string is unclosed, no error
}

// skipComment 跳过注释, 返回是否成功跳过
// 支持两种注释格式: -- 单行注释 和 /* */ 多行注释
// skipComment skips comments, returns whether it successfully skipped
// Supports two comment formats: -- single-line comment and /* */ multi-line comment
func (sc *sqlScanner) skipComment() bool {
	// -- comment: 单行注释 / Single-line comment
	if sc.i+1 < sc.n && (*sc.s)[sc.i] == '-' && (*sc.s)[sc.i+1] == '-' {
		sc.i += 2 // 跳过 -- / Skip --
		// 扫描到行尾或 EOF (EOF 也视为注释结束)
		// Scan to end of line or EOF (EOF also counts as end of comment)
		for sc.i < sc.n && (*sc.s)[sc.i] != '\n' {
			sc.i++
		}
		return true
	}

	// /* comment */: 多行注释 / Multi-line comment
	if sc.i+1 < sc.n && (*sc.s)[sc.i] == '/' && (*sc.s)[sc.i+1] == '*' {
		sc.i += 2 // 跳过 /* / Skip /*
		for sc.i+1 < sc.n {
			if (*sc.s)[sc.i] == '*' && (*sc.s)[sc.i+1] == '/' {
				sc.i += 2 // 跳过 */ / Skip */
				return true
			}
			sc.i++
		}
		// 注释未闭合也返回 true / Returns true even if comment is unclosed
		return true
	}

	return false // 不是注释 / Not a comment
}

// ================= 关键字匹配 / Keyword Matching =================

// matchKeyword 忽略大小写匹配关键字, 并检查单词边界
// 例如: 匹配 "from" 时不会匹配到 "from_addr" 或 "afrom"
// matchKeyword matches keywords case-insensitively and checks word boundaries
// For example: matching "from" will not match "from_addr" or "afrom"
func matchKeyword(s *string, i int, word string) bool {
	n := len(*s)
	wlen := len(word)

	// 长度检查 / Length check
	if i+wlen > n {
		return false
	}

	// 前边界检查: 前面的字符不能是标识符字符
	// Front boundary check: previous character must not be an identifier character
	if i > 0 && isIdentChar((*s)[i-1]) {
		return false
	}

	// 匹配内容 (忽略大小写)
	// Match content (case-insensitive)
	for j := 0; j < wlen; j++ {
		c := (*s)[i+j]
		if c >= 'A' && c <= 'Z' {
			c += 32 // 转换为小写 / Convert to lowercase
		}
		if c != word[j] {
			return false
		}
	}

	// 后边界检查: 后面的字符不能是标识符字符
	// Back boundary check: next character must not be an identifier character
	if i+wlen < n && isIdentChar((*s)[i+wlen]) {
		return false
	}

	return true
}

// matchTwoKeywords 匹配两个连续的关键字, 如 "group by" 或 "order by"
// 允许两个关键字之间有空格、制表符、换行符
// matchTwoKeywords matches two consecutive keywords like "group by" or "order by"
// Allows spaces, tabs, and newlines between the two keywords
func matchTwoKeywords(s *string, i int, w1, w2 string) bool {
	// 先匹配第一个关键字 / First match the first keyword
	if !matchKeyword(s, i, w1) {
		return false
	}

	j := i + len(w1)

	// 跳过中间的空白字符 (空格、制表符、换行符)
	// Skip whitespace between keywords (spaces, tabs, newlines)
	for j < len(*s) {
		switch (*s)[j] {
		case ' ', '\t', '\n', '\r':
			j++
		default:
			goto CHECK
		}
	}

CHECK:
	// 匹配第二个关键字 / Match the second keyword
	return matchKeyword(s, j, w2)
}

// ================= 核心解析 / Core Parsing =================

// parseSQL 解析 SQL 语句, 返回各个子句的位置片段
// 这是替代正则表达式方案的核心函数, 用于分页时包装 COUNT(*) 语句
// 特点:
//   - 单次扫描完成所有关键字解析, 性能优于多次正则匹配
//   - 正确处理括号嵌套 (子查询中的 FROM 不影响外层)
//   - 正确处理字符串和注释中的伪关键字
//   - 大小写不敏感
//
// parseSQL parses a SQL statement and returns position fragments for each clause
// This is the core function to replace the regex-based approach, used for wrapping COUNT(*) in pagination
// Features:
//   - Single scan completes all keyword parsing, better performance than multiple regex matches
//   - Correctly handles parentheses nesting (FROM in subquery doesn't affect outer query)
//   - Correctly handles pseudo-keywords in strings and comments
//   - Case-insensitive
func parseSQL(sql *string) sqlPart {
	// 使用局部变量存储字符串值, 避免频繁解引用
	// Use local variable to store string value and avoid frequent dereferencing
	s := *sql
	sc := &sqlScanner{s: &s, n: len(s)}
	var parts sqlPart
	current := &parts.Select // 当前正在解析的子句, 默认为 SELECT / Current clause being parsed, defaults to SELECT
	current.Start = 0        // SELECT 始终从位置 0 开始 / SELECT always starts at position 0

	for sc.i < sc.n {
		c := s[sc.i]

		// 1. 字符串字面量: 跳过整个字符串, 避免误解析字符串中的关键字
		// String literal: skip the entire string to avoid misparsing keywords inside
		if c == '\'' || c == '"' {
			sc.skipString()
			continue
		}

		// 2. 注释: 跳过注释内容
		// Comment: skip comment content
		if c == '-' || c == '/' {
			if sc.skipComment() {
				continue
			}
		}

		// 3. 括号深度管理: 用于处理子查询
		// Parentheses depth management: for handling subqueries
		switch c {
		case '(':
			sc.depth++ // 进入子查询 / Enter subquery
			sc.i++
			continue

		case ')':
			if sc.depth > 0 {
				sc.depth-- // 退出子查询 / Exit subquery
			}
			sc.i++
			continue
		}

		// 4. 只在最外层 (非子查询内) 解析关键字
		// Only parse keywords at the outermost level (not inside subqueries)
		if sc.depth == 0 {
			switch c {
			case 'f', 'F':
				if matchKeyword(sc.s, sc.i, "from") {
					current.End = sc.i      // 结束当前子句 / End current clause
					parts.From.Start = sc.i // 设置 FROM 起始位置 / Set FROM start position
					current = &parts.From   // 切换到 FROM 子句 / Switch to FROM clause
				}

			case 'w', 'W':
				if matchKeyword(sc.s, sc.i, "where") {
					current.End = sc.i       // 结束当前子句 / End current clause
					parts.Where.Start = sc.i // 设置 WHERE 起始位置 / Set WHERE start position
					current = &parts.Where   // 切换到 WHERE 子句 / Switch to WHERE clause
				}

			case 'g', 'G':
				if matchTwoKeywords(sc.s, sc.i, "group", "by") {
					current.End = sc.i         // 结束当前子句 / End current clause
					parts.GroupBy.Start = sc.i // 设置 GROUP BY 起始位置 / Set GROUP BY start position
					current = &parts.GroupBy   // 切换到 GROUP BY 子句 / Switch to GROUP BY clause
				}

			case 'o', 'O':
				if matchTwoKeywords(sc.s, sc.i, "order", "by") {
					current.End = sc.i         // 结束当前子句 / End current clause
					parts.OrderBy.Start = sc.i // 设置 ORDER BY 起始位置 / Set ORDER BY start position
					current = &parts.OrderBy   // 切换到 ORDER BY 子句 / Switch to ORDER BY clause
				}

			}
		}

		sc.i++
	}

	// 设置最后一个子句的结束位置
	// Set end position for the last clause
	if current != nil {
		current.End = sc.n
	}

	// 为所有已启动但未设置 End 的 part 补全 End 值
	// Complete End values for parts that were started but not ended
	// 使用 Start > 0 判断, 因为正常 SQL 中这些关键字不可能在位置 0
	// Uses Start > 0 check because these keywords cannot be at position 0 in normal SQL
	if parts.From.Start > 0 && parts.From.End == 0 {
		parts.From.End = sc.n
	}
	if parts.Where.Start > 0 && parts.Where.End == 0 {
		parts.Where.End = sc.n
	}
	if parts.GroupBy.Start > 0 && parts.GroupBy.End == 0 {
		parts.GroupBy.End = sc.n
	}
	if parts.OrderBy.Start > 0 && parts.OrderBy.End == 0 {
		parts.OrderBy.End = sc.n
	}

	return parts
}
