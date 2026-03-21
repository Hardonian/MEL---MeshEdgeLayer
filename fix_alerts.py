#!/usr/bin/env python3

# Update alerts.go
with open('internal/service/alerts.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Add import for selfobs
if 'selfobs' not in content:
    old = '''import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	statuspkg "github.com/mel-project/mel/internal/status"
)'''
    new = '''import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/selfobs"
	statuspkg "github.com/mel-project/mel/internal/status"
)'''
    content = content.replace(old, new)
    with open('internal/service/alerts.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Updated alerts.go imports')

# Add MarkFresh call at the end of evaluateTransportIntelligence function
with open('internal/service/alerts.go', 'r', encoding='utf-8') as f:
    content = f.read()

old2 = '''	a.evaluateControl(now)
}

func deriveTransportAlerts'''

new2 = '''	a.evaluateControl(now)
	selfobs.MarkFresh("alert")
}

func deriveTransportAlerts'''

if old2 in content:
    content = content.replace(old2, new2)
    with open('internal/service/alerts.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Added MarkFresh to alerts.go')
else:
    print('alerts.go pattern not found')
