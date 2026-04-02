import os

# Third pass: catch all remaining Chinese fragments
translations = {
    # DataTable remaining
    '每页行数': 'Rows per page',
    '页面大小必须在': 'Page size must be between',
    '到': 'and',
    '之间': '',
    '共 ': 'Total: ',
    '条记录': 'records',
    '已选择': 'Selected',
    '项': 'items',
    '加载': 'Loading',
    '失败': 'failed',
    'Custom页面大小': 'Custom page size',
    
    # api/endpoint.ts
    '批量Test Endpoint': 'Batch test endpoints',
    '批量Delete Endpoint': 'Batch delete endpoints',
    '手动触发Endpoints测试': 'Manually trigger endpoint test',
    '获取Endpoints测试结果': 'Get endpoint test results',
    
    # api/index.ts
    '导出所有 API 服务': 'Export all API services',
    '导出所有API服务': 'Export all API services',
    
    # api/plan.ts
    '获取所有Plan': 'Get all plans',
    '根据 ID 获取Plan': 'Get plan by ID',
    '获取当前User的Plan': 'Get current user plan',
    
    # api/setting.ts
    '获取所有Settings': 'Get all settings',
    '根据 key 获取Settings': 'Get setting by key',
    
    # LoadingFallback
    '用于代码分割加载Status的通用加载组件': 'Generic loading component for code-split loading states',
    
    # ProtectedRoute
    '加载中，可以返回一个加载组件': 'Loading, return a loading component',
    '如果User未认证，重定向到Sign In页面': 'If user is not authenticated, redirect to login',
    '如果需要Admin权限，但User不YesAdmin，重定向到Home或显示无权限页面': 'If admin required but user is not admin, redirect to unauthorized page',
    'User已认证且满足权限要求，渲染子路由': 'User is authenticated and meets permission requirements, render child routes',
    
    # SearchForm
    '自动搜索延迟（毫秒）': 'Auto-search delay (ms)',
    '默认不自动搜索': 'No auto-search by default',
    '当外部的 searchTerm 改变时，Update本地的 searchTerm': 'When external searchTerm changes, update local searchTerm',
    'Handle search输入变化并支持自动搜索': 'Handle search input change with auto-search support',
    '如果Settings了自动搜索延迟，启用防抖自动搜索': 'If auto-search delay is set, enable debounced auto-search',
    
    # CreateModal
    '成功Create': 'Successfully created',
    '个Endpoints': ' endpoints',
    
    # api/endpoint.ts remaining
    '获取所有Endpoints（带最近Performance测试和AI Models数量）': 'Get all endpoints (with recent performance tests and AI model counts)',
    '获取单个Endpoints详情（包含AI Models）': 'Get single endpoint details (with AI models)',
    
    # hooks
    '验证配置类型': 'Validation config type',
    
    # types
    '分页参数': 'Pagination params',
    '分页响应': 'Paginated response',
    '排序顺序': 'Sort order',
    '通用': 'Generic',
}

def translate_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    original = content
    
    sorted_t = sorted(translations.items(), key=lambda x: len(x[0]), reverse=True)
    for chinese, english in sorted_t:
        content = content.replace(chinese, english)
    
    if content != original:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        return True
    return False

count = 0
for root, dirs, files in os.walk('frontend/src'):
    for f in files:
        if f.endswith(('.tsx', '.ts')):
            path = os.path.join(root, f)
            if translate_file(path):
                count += 1
                print(f'Pass 3 translated: {path}')

print(f'\nPass 3 total: {count}')
