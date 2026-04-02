import os, re

# Second pass: translate remaining mixed Chinese/English comments
translations_pass2 = {
    # App.tsx
    '// 使用 React.lazy 动态导入页面组件': '// Lazy-load page components with React.lazy',
    '{/* 公共路由 */}': '{/* Public routes */}',
    '{/* 受保护路由 */}': '{/* Protected routes */}',
    '{/* Endpoints相关路由 */}': '{/* Endpoint routes */}',
    '{/* Model相关路由 */}': '{/* Model routes */}',
    '{/* API Keys相关路由 */}': '{/* API Key routes */}',
    '{/* UserProfile和Settings路由 */}': '{/* Profile and Settings routes */}',
    '{/* Admin路由 */}': '{/* Admin routes */}',
    '{/* 404 页面 */}': '{/* 404 page */}',
    
    # api/aimodel.ts
    '// 获取所有 AI Models（带最近Performance测试）': '// Get all AI models (with recent performance tests)',
    '// 获取单个 AI Models详情（包含Endpoints）': '// Get single AI model details (with endpoints)',
    
    # api/apikey.ts
    '// 获取当前User的所有 API Keys': '// Get all API keys for current user',
    '// Create新的 API Keys': '// Create new API key',
    '// 获取 API KeysUsage Statistics': '// Get API key usage statistics',
    '// 删除 API Keys': '// Delete API key',
    
    # api/auth.ts
    '// 初始化第一个User': '// Initialize first user',
    '// 获取当前User信息': '// Get current user info',
    '// 修改当前UserPassword': '// Change current user password',
    '// Create New User (需要Admin权限)': '// Create new user (requires admin privileges)',
    '// 获取所有User (需要Admin权限)': '// Get all users (requires admin privileges)',
    '// 根据 ID 获取User (需要Admin权限)': '// Get user by ID (requires admin privileges)',
    '// UpdateUser (需要Admin权限)': '// Update user (requires admin privileges)',
    '// DeleteUser (需要Admin权限)': '// Delete user (requires admin privileges)',
    
    # api/client.ts
    '// 定义 API 响应的基本结构': '// Define base API response structure',
    '// 扩展AxiosError接口以包含detail字段': '// Extend AxiosError interface to include detail field',
    '// 将查询参数对象转换为URL查询字符串': '// Convert query params object to URL query string',
    '// 请求拦截器 - Add认证 token': '// Request interceptor - add auth token',
    '// 响应拦截器 - 处理错误和刷新 token': '// Response interceptor - handle errors and refresh token',
    '// 处理错误响应中的detail字段': '// Handle detail field in error response',
    '// 检查YesNo包含detail字段': '// Check if response contains detail field',
    '// 处理 401 错误（未授权）': '// Handle 401 error (unauthorized)',
    '// 如果需要实现 token 刷新，可以在这里Add逻辑': '// If token refresh is needed, add logic here',
    '// 当前简单实现：401 就清除 token 并跳转到Sign In页': '// Current simple implementation: clear token on 401 and redirect to login',
    '// 通用 GET 请求': '// Generic GET request',
    '// 通用 POST 请求': '// Generic POST request',
    '// 通用 PUT 请求': '// Generic PUT request',
    '// 通用 PATCH 请求': '// Generic PATCH request',
    '// 通用 DELETE 请求': '// Generic DELETE request',
    '// Create默认客户端实例': '// Create default client instance',
    
    # api/endpoint.ts
    '// 获取所有Endpoints（带最近Performance测试和 AI Models数量）': '// Get all endpoints (with recent performance tests and AI model counts)',
    '// 获取单个Endpoints详情（包含 AI Models）': '// Get single endpoint details (with AI models)',
    '// Create新的Endpoints': '// Create new endpoint',
    '// Batch Create或更新Endpoints': '// Batch create or update endpoints',
    '// 批量UpdateEndpoints': '// Batch update endpoints',
    '// DeleteEndpoints': '// Delete endpoint',
    '// Batch DeleteEndpoints': '// Batch delete endpoints',
    '// TestEndpoints': '// Test endpoint',
    '// Batch TestEndpoints': '// Batch test endpoints',
    '// 获取Endpoints测试任务Status': '// Get endpoint test task status',
    
    # api/plan.ts
    '// 获取所有Plans': '// Get all plans',
    '// Create新Plan': '// Create new plan',
    '// UpdatePlan': '// Update plan',
    '// DeletePlan': '// Delete plan',
    
    # api/setting.ts
    '// 获取SystemSettings': '// Get system setting',
    '// UpdateSystemSettings': '// Update system setting',
    
    # api/index.ts
    '// 导出所有API服务': '// Export all API services',
    
    # components
    '// 加载状态组件': '// Loading state component',
    '// 搜索表单组件': '// Search form component',
    '// 受保护路由需要认证': '// Protected route requires authentication',
    '// 查询提供者组件': '// Query provider component',
    '// URL状态管理': '// URL state management',
    
    # types
    '// 通用分页参数': '// Generic pagination params',
    '// 分页响应': '// Paginated response',
    '// 排序顺序': '// Sort order',
    '// 批量操作结果': '// Batch operation result',
}

def translate_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    original = content
    
    sorted_translations = sorted(translations_pass2.items(), key=lambda x: len(x[0]), reverse=True)
    for chinese, english in sorted_translations:
        content = content.replace(chinese, english)
    
    # Generic fallback: translate any remaining Chinese-only comment lines
    # Match lines that contain only Chinese comment text
    def replace_remaining_chinese(match):
        line = match.group(0)
        # Just return as-is - we'll handle stragglers individually
        return line
    
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
                print(f'Pass 2 translated: {path}')

print(f'\nPass 2 total: {count}')
