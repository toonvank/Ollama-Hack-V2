import os

replacements = {
    '// Endpoint Performance Information': '// Endpoint Performance Information',
    '// Endpoint Info with AI Models count': '// Endpoint Info with AI Models count',
    '// Endpoint AI Models Information': '// Endpoint AI Models Information',
    '// Endpoint Info with AI Models list': '// Endpoint Info with AI Models list',
    '// Endpoint Create Request': '// Endpoint Create Request',
    '// Endpoint Update Request': '// Endpoint Update Request',
    '// Batch Create Endpoints Request': '// Batch Create Endpoints Request',
    '// Batch Action Endpoints Request': '// Batch Action Endpoints Request',
    '// List of endpoint IDs to perform action on': '// List of endpoint IDs to perform action on',
    '// Batch Action Results Response': '// Batch Action Results Response',
    '// Number of successfully processed endpoints': '// Number of successfully processed endpoints',
    '// Number of failed endpoints': '// Number of failed endpoints',
    '// Failed endpoint IDs and reasons': '// Failed endpoint IDs and reasons',
    '// Endpoint Task Information': '// Endpoint Task Information',
    '// Export all types': '// Export all types',
    '// AI Models Performance Information': '// AI Models Performance Information',
    '// Model Info with Endpoint Info': '// Model Info with Endpoint Info',
    '// AI Models Info with Endpoint Count': '// AI Models Info with Endpoint Count',
    '// AI Models details with Endpoints': '// AI Models details with Endpoints',
    '// Plan Create Request': '// Plan Create Request',
    '// Plan Response': '// Plan Response',
    '// Plan Update Request': '// Plan Update Request',
    '# AI model schemas': '# AI model schemas',
    '# Add search conditions': '# Add search conditions',
    '# Add search conditions': '# Add search conditions',
    '# Add basic sorting': '# Add basic sorting',
    '# Add sorting': '# Add sorting',
    '# Add sorting': '# Add sorting',
    '"server busy"': '"server busy"',
}

for root, _, files in os.walk('.'):
    if 'node_modules' in root or '.git' in root or '.gemini' in root:
        continue
    for file in files:
        if file.endswith('.ts') or file.endswith('.tsx') or file.endswith('.py') or file.endswith('.tsx'):
            path = os.path.join(root, file)
            with open(path, 'r', encoding='utf-8') as f:
                content = f.read()
            original = content
            for k, v in replacements.items():
                content = content.replace(k, v)
            if content != original:
                with open(path, 'w', encoding='utf-8') as f:
                    f.write(content)
                print(f"Updated {path}")
