import os
import re

# Additional translations for remaining Chinese text
additional_translations = {
    # Newly found strings
    '详情': 'Details',
    '测试已开始': 'Test started',
    '请等待结果': 'Please wait for results',
    '实例': 'instance',
    '分钟': 'minutes',
    '大屏幕下显示的主题切换开关': 'Theme toggle for large screens',
    '菜单': 'Menu',
    '小屏幕下显示的主题切换选': 'Theme toggle for small screens',
    '清空表单': 'Clear form',
    '最后使用时间': 'Last Used',
    '对话框': 'Dialog',
    '密钥统计抽屉': 'Key statistics drawer',
    '抽屉': 'Drawer',
    '聚合': 'Aggregated',
    '统计信息': 'Statistics',
    '用于': 'for',
    '的': 'of',
    '对象': 'object',
    '将': 'will',
    '详情抽屉': 'Details drawer',
    
    # Common fragments found in useUrlState.ts
    '使用': 'Use',
    '参数管理': 'parameter management',
    '参数配置': 'parameter configuration',
    '和': 'and',
    '自定义': 'Custom',
    '钩子': 'Hook',
    '状态': 'state',
    '分页': 'pagination',
    '排序': 'sorting',
    '搜索': 'search',
    '字段': 'field',
    '顺序': 'order',
    '验证': 'validation',
    '配置': 'configuration',
    '最小值': 'minimum',
    '最大值': 'maximum',
    '默认值': 'default',
    '允许': 'allowed',
    '字段列表': 'field list',
    '排序字段': 'sort field',
    '升序': 'ascending',
    '降序': 'descending',
    '总页数': 'total pages',
    '当前页': 'current page',
    '每页': 'per page',
    '大小': 'size',
    '参数': 'parameter',
    '值': 'value',
    '更新': 'update',
    '设置': 'set',
}

def translate_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    original = content
    
    # Apply translations (longer first to avoid partial matches)
    sorted_translations = sorted(additional_translations.items(), key=lambda x: len(x[0]), reverse=True)
    for chinese, english in sorted_translations:
        # Use word boundary for single character translations to avoid false matches
        if len(chinese) == 1:
            # Only replace if surrounded by non-Chinese characters or at boundaries
            pattern = re.compile(r'(?<![\\u4e00-\\u9fff])' + re.escape(chinese) + r'(?![\\u4e00-\\u9fff])')
            content = pattern.sub(english, content)
        else:
            content = content.replace(chinese, english)
    
    if content != original:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        return True
    return False

# Process all frontend files
count = 0
chinese_pattern = re.compile(r'[\u4e00-\u9fff]+')

for root, dirs, files in os.walk('frontend/src'):
    for f in files:
        if f.endswith(('.tsx', '.ts')):
            path = os.path.join(root, f)
            try:
                with open(path, 'r', encoding='utf-8') as file:
                    content = file.read()
                    if chinese_pattern.search(content):
                        if translate_file(path):
                            count += 1
                            print(f'Translated: {path}')
                        else:
                            # Still has Chinese after translation
                            remaining = chinese_pattern.findall(content)
                            if remaining:
                                print(f'Still has Chinese: {path}')
                                print(f'  Remaining: {set(remaining)}')
            except Exception as e:
                print(f'Error processing {path}: {e}')

print(f'\nTotal files translated: {count}')
