#!/usr/bin/env python3
with open('cmd/mel/main.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Add health to usage function - between db vacuum and ui
old = '  db vacuum --config <path>\n  ui --config <path>'
new = '  db vacuum --config <path>\n  health internal|freshness|slo|metrics --config <path>\n  ui --config <path>'

if old in content:
    content = content.replace(old, new)
    with open('cmd/mel/main.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Added health to usage')
else:
    print('Pattern not found')
    # Let me search for any patterns
    if 'db vacuum' in content:
        print('db vacuum found')
