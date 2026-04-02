import re

# File-specific fixes
fixes = {
    'frontend/src/hooks/useUrlState.ts': [
        ('UseURLparameter managementStatus的Hook', 'Hook for URL parameter state management'),
        ('Status和Update函数', 'State and update functions'),
        ('从URLparameter获取初始value，如果不存在则UseinitialState', 'Get initial value from URL parameter, use initialState if not present'),
        ('UpdateStatus时同步andURLparameter', 'Sync URL parameter when updating state'),
        ('忽略错误', 'Ignore errors'),
    ],
    'frontend/src/pages/apikeys/index.tsx': [
        ('会智能检测你Use的Model，并按照生成速度order尝试最优Endpoints转发你of请求。所有of', 'Intelligently detects the models you use and forwards your requests to optimal endpoints based on generation speed. All'),
        ('API 都and Ollama of API', 'APIs are compatible with Ollama APIs'),
        ('兼容，包括流式生成等功能。你只需要按照 Ollama of API', ', including streaming generation. Simply follow the Ollama API'),
        ('文档来调用即可，下面提供了 Ollama 官方 API 文档以供参考。', 'documentation to make calls. Official Ollama API documentation is provided below for reference.'),
    ],
    'frontend/src/pages/dashboard/index.tsx': [
        ('will ApiError 转换为 Error object', 'Convert ApiError to Error object'),
        ('👋 你好, {user?.username}', '👋 Hello, {user?.username}'),
        ('已注册User总数', 'Total registered users'),
        ('已注册User', 'Registered users'),
    ],
    'frontend/src/pages/users/index.tsx': [
        ('validation表单', 'Validate form'),
    ],
    'frontend/src/types/apikey.ts': [
        ('// API Key Creation请求', '// API Key creation request'),
        ('// API Keys信息', '// API Key info'),
        ('// API Keys响应（包含密钥value）', '// API Key response (includes key value)'),
    ],
    'frontend/src/types/common.ts': [
        ('// Paginated responseGeneric类型', '// Paginated response generic type'),
        ('// API 请求Genericparameter', '// API request generic parameters'),
        ('// sorting方向枚举', '// Sort direction enum'),
        ('// Generic查询parameter接口', '// Generic query parameter interface'),
        ('// API 错误响应', '// API error response'),
    ],
}

for filepath, replacements in fixes.items():
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original = content
        for old, new in replacements:
            content = content.replace(old, new)
        
        if content != original:
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(content)
            print(f"Fixed: {filepath}")
    except Exception as e:
        print(f"Error processing {filepath}: {e}")

print("\nDone!")
