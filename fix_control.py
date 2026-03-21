#!/usr/bin/env python3

# Update control.go
with open('internal/service/control.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Add import for selfobs
if 'selfobs' not in content:
    old = '''import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
)'''
    new = '''import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mel-project/mel/internal/control"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/selfobs"
)'''
    content = content.replace(old, new)
    with open('internal/service/control.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Updated control.go imports')

# Add MarkFresh call after control action completes successfully
with open('internal/service/control.go', 'r', encoding='utf-8') as f:
    content = f.read()

old2 = '''	if result == control.ResultExecutedSuccessfully && (action.ActionType == control.ActionRestartTransport || action.ActionType == control.ActionResubscribeTransport) {
		a.enqueueHealthRecheckFollowup(action)
	}
}

func (a *App) findTransportAndControl'''

new2 = '''	if result == control.ResultExecutedSuccessfully && (action.ActionType == control.ActionRestartTransport || action.ActionType == control.ActionResubscribeTransport) {
		a.enqueueHealthRecheckFollowup(action)
	}
	if result == control.ResultExecutedSuccessfully {
		selfobs.MarkFresh("control")
	}
}

func (a *App) findTransportAndControl'''

if old2 in content:
    content = content.replace(old2, new2)
    with open('internal/service/control.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Added MarkFresh to control.go')
else:
    print('control.go pattern not found')
