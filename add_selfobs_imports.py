#!/usr/bin/env python3

# 1. Update ingest.go
with open('internal/db/ingest.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Add import for selfobs
old = '''import (
	"encoding/json"
	"fmt"
	"strings"
)'''

new = '''import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mel-project/mel/internal/selfobs"
)'''

if old in content and 'selfobs' not in content:
    content = content.replace(old, new)
    with open('internal/db/ingest.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Updated ingest.go imports')
else:
    print('ingest.go already has selfobs or pattern not found')

# Add MarkFresh call after successful persist
with open('internal/db/ingest.go', 'r', encoding='utf-8') as f:
    content = f.read()

old2 = '''	return asInt(rows[0]["message_inserted"]) > 0, nil
}'''

new2 = '''	if asInt(rows[0]["message_inserted"]) > 0 {
		selfobs.MarkFresh("ingest")
	}
	return asInt(rows[0]["message_inserted"]) > 0, nil
}'''

if old2 in content:
    content = content.replace(old2, new2)
    with open('internal/db/ingest.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Added MarkFresh to ingest.go')
else:
    print('ingest.go pattern not found')
