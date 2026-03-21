#!/usr/bin/env python3
with open('cmd/mel/main.go', 'r', encoding='utf-8') as f:
    lines = f.readlines()

# Find line index for "func serveCmd"
target_idx = None
for i, line in enumerate(lines):
    if 'func serveCmd(args []string) {' in line:
        target_idx = i
        break

if target_idx:
    # Insert the healthCmd function before serveCmd
    health_func = '''func healthCmd(args []string) {
\tif len(args) == 0 {
\t\tpanic("usage: mel health internal|freshness|slo|metrics --config <path>")
\t}
\tswitch args[0] {
\tcase "internal":
\t\thealthInternalCmd(args[1:])
\tcase "freshness":
\t\thealthFreshnessCmd(args[1:])
\tcase "slo":
\t\thealthSLOCmd(args[1:])
\tcase "metrics":
\t\thealthMetricsCmd(args[1:])
\tdefault:
\t\tpanic("usage: mel health internal|freshness|slo|metrics --config <path>")
\t}
}

'''
    lines.insert(target_idx, health_func)
    with open('cmd/mel/main.go', 'w', encoding='utf-8') as f:
        f.writelines(lines)
    print(f'Inserted at line {target_idx + 1}')
else:
    print('Target not found')
