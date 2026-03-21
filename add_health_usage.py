#!/usr/bin/env python3
with open('cmd/mel/main.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Add health to usage function
old = '  diagnostics --config <path>\n  dev-simulate-mqtt'
new = '  diagnostics --config <path>\n  health internal|freshness|slo|metrics --config <path>\n  dev-simulate-mqtt'

if old in content:
    content = content.replace(old, new)
    with open('cmd/mel/main.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Added health to usage')
else:
    print('Pattern not found')
