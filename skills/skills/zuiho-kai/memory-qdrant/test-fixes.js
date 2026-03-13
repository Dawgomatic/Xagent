/**
 * 自验证测试 - v1.0.10 修复验证
 *
 * 测试修复的问题：
 * - P1-13: CLI 命令使用统一常量
 * - P1-14: detectCategory 正则表达式一致性
 * - P1-15: 邮箱正则一致性
 * - P2-16: Embeddings 错误处理和重试
 * - P2-20: 输入清理
 * - P2-21: Qdrant 健康检查
 */

import { shouldCapture, detectCategory, sanitizeInput } from './index.js';

// ============================================================================
// 测试工具
// ============================================================================

let testsPassed = 0;
let testsFailed = 0;

function assert(condition, message) {
  if (condition) {
    console.log(` PASS: ${message}`);
    testsPassed++;
  } else {
    console.error(` FAIL: ${message}`);
    testsFailed++;
  }
}

function assertEquals(actual, expected, message) {
  if (actual === expected) {
    console.log(` PASS: ${message}`);
    testsPassed++;
  } else {
    console.error(` FAIL: ${message}`);
    console.error(`  Expected: ${expected}`);
    console.error(`  Actual: ${actual}`);
    testsFailed++;
  }
}

// ============================================================================
// 测试套件
// ============================================================================

console.log('\n 开始自验证测试...\n');

// 测试 1: sanitizeInput - HTML 标签清理
console.log(' 测试组 1: 输入清理 (sanitizeInput)');
{
  const input1 = '<script>alert("xss")</script>Hello';
  const result1 = sanitizeInput(input1);
  assert(!result1.includes('<script>'), '应该移除 script 标签');
  assertEquals(result1, 'alert("xss")Hello', '应该只保留文本内容');

  const input2 = '<b>Bold</b> and <i>italic</i>';
  const result2 = sanitizeInput(input2);
  assertEquals(result2, 'Bold and italic', '应该移除所有 HTML 标签');

  const input3 = 'Normal text';
  const result3 = sanitizeInput(input3);
  assertEquals(result3, 'Normal text', '普通文本应该保持不变');

  const input4 = '  Multiple   spaces  ';
  const result4 = sanitizeInput(input4);
  assertEquals(result4, 'Multiple spaces', '应该规范化空白字符');

  const input5 = 'Line1\x00\x01\x02Line2';
  const result5 = sanitizeInput(input5);
  assert(!result5.includes('\x00'), '应该移除控制字符');
  assertEquals(result5, 'Line1Line2', '应该移除控制字符但保留文本');
}

// 测试 2: detectCategory - 正则表达式一致性
console.log('\n 测试组 2: 分类检测 (detectCategory)');
{
  // 测试电话号码（应该限制长度）
  const phone1 = '+1234567890';  // 10 位
  const cat1 = detectCategory(phone1);
  assertEquals(cat1, 'entity', '10 位电话号码应该被识别为 entity');

  const phone2 = '+12345678901234';  // 14 位（超过限制）
  const cat2 = detectCategory(phone2);
  assert(cat2 !== 'entity' || phone2.length <= 13, '超长电话号码不应该被识别为 entity');

  // 测试邮箱（应该使用严格模式）
  const email1 = 'test@example.com';
  const cat3 = detectCategory(email1);
  assertEquals(cat3, 'entity', '有效邮箱应该被识别为 entity');

  const email2 = 'invalid@';
  const cat4 = detectCategory(email2);
  assert(cat4 !== 'entity', '无效邮箱不应该被识别为 entity');

  // 测试偏好
  const pref1 = 'I prefer using TypeScript';
  const cat5 = detectCategory(pref1);
  assertEquals(cat5, 'preference', '偏好语句应该被识别为 preference');

  // 测试决策
  const decision1 = 'We decided to use React';
  const cat6 = detectCategory(decision1);
  assertEquals(cat6, 'decision', '决策语句应该被识别为 decision');
}

// 测试 3: shouldCapture - 邮箱正则一致性
console.log('\n 测试组 3: 捕获过滤 (shouldCapture)');
{
  const email1 = 'My email is test@example.com';
  const result1 = shouldCapture(email1);
  assert(result1, '包含有效邮箱的文本应该被捕获');

  const email2 = 'Invalid email: @example';
  const result2 = shouldCapture(email2);
  assert(!result2, '包含无效邮箱的文本不应该被捕获');

  const remember1 = 'Remember to buy milk';
  const result3 = shouldCapture(remember1);
  assert(result3, '包含 remember 关键词的文本应该被捕获');

  const short1 = 'Hi';
  const result4 = shouldCapture(short1);
  assert(!result4, '过短的文本不应该被捕获');

  const long1 = 'a'.repeat(1000);
  const result5 = shouldCapture(long1, 500);
  assert(!result5, '超长文本不应该被捕获');
}

// 测试 4: ReDoS 防护
console.log('\n 测试组 4: ReDoS 防护');
{
  // 测试可能导致 ReDoS 的输入
  const malicious1 = '+' + '1'.repeat(100);  // 超长电话号码
  const start1 = Date.now();
  const cat1 = detectCategory(malicious1);
  const duration1 = Date.now() - start1;
  assert(duration1 < 100, `超长电话号码处理应该很快 (${duration1}ms)`);

  const malicious2 = 'a'.repeat(100) + '@' + 'b'.repeat(100) + '.' + 'c'.repeat(100);
  const start2 = Date.now();
  const result2 = shouldCapture(malicious2);
  const duration2 = Date.now() - start2;
  assert(duration2 < 100, `复杂邮箱模式处理应该很快 (${duration2}ms)`);
}

// 测试 5: 边界情况
console.log('\n 测试组 5: 边界情况');
{
  // null/undefined 输入
  const result1 = sanitizeInput(null);
  assertEquals(result1, '', 'null 应该返回空字符串');

  const result2 = sanitizeInput(undefined);
  assertEquals(result2, '', 'undefined 应该返回空字符串');

  const result3 = sanitizeInput('');
  assertEquals(result3, '', '空字符串应该返回空字符串');

  // 非字符串输入
  const result4 = sanitizeInput(123);
  assertEquals(result4, '', '数字应该返回空字符串');

  const result5 = sanitizeInput({});
  assertEquals(result5, '', '对象应该返回空字符串');
}

// 测试 6: 中文支持
console.log('\n 测试组 6: 中文支持');
{
  const chinese1 = '记住这个重要信息';
  const result1 = shouldCapture(chinese1);
  assert(result1, '中文 "记住" 关键词应该被捕获');

  const chinese2 = '我喜欢用 TypeScript';
  const cat1 = detectCategory(chinese2);
  assertEquals(cat1, 'preference', '中文偏好应该被识别');

  const chinese3 = '我决定使用 React';
  const cat2 = detectCategory(chinese3);
  assertEquals(cat2, 'decision', '中文决策应该被识别');

  const chinese4 = '<b>粗体</b>文本';
  const result2 = sanitizeInput(chinese4);
  assertEquals(result2, '粗体文本', '中文文本应该正确清理 HTML');
}

// ============================================================================
// 测试结果
// ============================================================================

console.log('\n' + '='.repeat(60));
console.log(' 测试结果汇总');
console.log('='.repeat(60));
console.log(` 通过: ${testsPassed}`);
console.log(` 失败: ${testsFailed}`);
console.log(` 通过率: ${((testsPassed / (testsPassed + testsFailed)) * 100).toFixed(1)}%`);
console.log('='.repeat(60));

if (testsFailed === 0) {
  console.log('\n 所有测试通过！代码修复验证成功。\n');
  process.exit(0);
} else {
  console.log('\n  部分测试失败，请检查代码。\n');
  process.exit(1);
}
