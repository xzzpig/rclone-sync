
const fs = require('fs');
const path = require('path');

const enPath = './web/project.inlang/messages/en.json';
const zhPath = './web/project.inlang/messages/zh-CN.json';

const en = JSON.parse(fs.readFileSync(enPath, 'utf8'));
const zh = JSON.parse(fs.readFileSync(zhPath, 'utf8'));

const enMap = new Map();
const zhMap = new Map();

// Map value to list of keys
for (const [key, value] of Object.entries(en)) {
    if (!enMap.has(value)) enMap.set(value, []);
    enMap.get(value).push(key);
}

for (const [key, value] of Object.entries(zh)) {
    if (!zhMap.has(value)) zhMap.set(value, []);
    zhMap.get(value).push(key);
}

// Find groups of keys that are identical in both
const potentialMerges = [];

// Iterate over keys in EN
const keys = Object.keys(en);
const processed = new Set();

for (const key1 of keys) {
    if (processed.has(key1)) continue;
    
    const valEn = en[key1];
    const valZh = zh[key1];
    
    if (!valZh) continue; // Key missing in ZH

    // Find other keys that have same EN value AND same ZH value
    const candidates = enMap.get(valEn).filter(k => k !== key1 && zh[k] === valZh);
    
    if (candidates.length > 0) {
        const group = [key1, ...candidates];
        group.forEach(k => processed.add(k));
        potentialMerges.push({
            keys: group,
            en: valEn,
            zh: valZh
        });
    }
}

console.log(JSON.stringify(potentialMerges, null, 2));
