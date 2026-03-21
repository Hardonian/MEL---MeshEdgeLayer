#!/usr/bin/env python3

# Update backup.go
with open('internal/backup/backup.go', 'r', encoding='utf-8') as f:
    content = f.read()

# Add import for selfobs
if 'selfobs' not in content:
    old = '''import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
)'''
    new = '''import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/selfobs"
)'''
    content = content.replace(old, new)
    with open('internal/backup/backup.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Updated backup.go imports')

# Add MarkFresh call after Create function completes
with open('internal/backup/backup.go', 'r', encoding='utf-8') as f:
    content = f.read()

old2 = '''	}
	return manifest, nil
}

func ValidateRestore'''

new2 = '''	}
	selfobs.MarkFresh("backup")
	return manifest, nil
}

func ValidateRestore'''

if old2 in content:
    content = content.replace(old2, new2)
    with open('internal/backup/backup.go', 'w', encoding='utf-8') as f:
        f.write(content)
    print('Added MarkFresh to backup.go')
else:
    print('backup.go pattern not found')
