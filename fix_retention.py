#!/usr/bin/env python3

# Update retention.go
with open('internal/db/retention.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Add import for selfobs
if 'selfobs' not in content:
    old = '''import (
	"fmt"
	"time"
)'''
    new = '''import (
	"fmt"
	"time"

	"github.com/mel-project/mel/internal/selfobs"
)'''
    content = content.replace(old, new)
    with open('internal/db/retention.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Updated retention.go imports')

# Add MarkFresh call after PruneTransportIntelligence
with open('internal/db/retention.go', 'r', encoding='utf-8') as f:
    content = f.read()

old2 = '''	return d.Exec(sql)
}

func (d *DB) PruneControlHistory'''

new2 = '''	if err := d.Exec(sql); err != nil {
		return err
	}
	selfobs.MarkFresh("retention")
	return nil
}

func (d *DB) PruneControlHistory'''

if old2 in content:
    content = content.replace(old2, new2)
    with open('internal/db/retention.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Added MarkFresh to retention.go')
else:
    print('retention.go pattern not found')
