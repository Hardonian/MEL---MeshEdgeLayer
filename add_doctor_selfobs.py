#!/usr/bin/env python3
with open('cmd/mel/main.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Add selfobs output to doctorCmd - after mustPrint(out) but before the exit check
old = '''	mustPrint(out)
	if len(findings) > 0 {
		os.Exit(1)
	}
}

func statusCmd(args []string) {'''

new = '''	mustPrint(out)

	// Add self-observability output
	fmt.Println()
	fmt.Println("=== Self-Observability ===")
	printLocalHealth()
	fmt.Println()
	printLocalFreshness()
	fmt.Println()
	printLocalSLO()

	if len(findings) > 0 {
		os.Exit(1)
	}
}

func statusCmd(args []string) {'''

if old in content:
    content = content.replace(old, new)
    with open('cmd/mel/main.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Added selfobs to doctor')
else:
    print('Pattern not found')
