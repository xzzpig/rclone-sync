#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

/**
 * 对 JSON 对象的 key 进行字典序排序
 * @param {Object} obj - 要排序的 JSON 对象
 * @returns {Object} 排序后的 JSON 对象
 */
function sortObjectKeys(obj) {
  const sorted = {};
  const keys = Object.keys(obj).sort();
  
  for (const key of keys) {
    sorted[key] = obj[key];
  }
  
  return sorted;
}

/**
 * 处理单个 JSON 文件
 * @param {string} filePath - 文件路径
 */
function sortJsonFile(filePath) {
  console.log(`处理文件: ${filePath}`);
  
  try {
    // 读取文件
    const content = fs.readFileSync(filePath, 'utf8');
    
    // 解析 JSON
    const json = JSON.parse(content);
    
    // 排序 keys
    const sorted = sortObjectKeys(json);
    
    // 写回文件 (保持格式，2 空格缩进)
    fs.writeFileSync(filePath, JSON.stringify(sorted, null, 2) + '\n', 'utf8');
    
    console.log(`✅ 成功排序: ${filePath}`);
    console.log(`   共 ${Object.keys(sorted).length} 个键\n`);
  } catch (error) {
    console.error(`❌ 处理文件失败 ${filePath}:`, error.message);
    process.exit(1);
  }
}

// 主函数
function main() {
  const files = [
    'web/project.inlang/messages/zh-CN.json',
    'web/project.inlang/messages/en.json'
  ];
  
  console.log('开始对 i18n JSON 文件的 key 进行字典序排序...\n');
  
  for (const file of files) {
    const filePath = path.join(process.cwd(), file);
    
    if (!fs.existsSync(filePath)) {
      console.error(`❌ 文件不存在: ${filePath}`);
      process.exit(1);
    }
    
    sortJsonFile(filePath);
  }
  
  console.log('✅ 所有文件处理完成！');
}

// 运行
main();
