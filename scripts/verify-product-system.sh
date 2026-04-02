#!/usr/bin/env bash
set -euo pipefail

python - <<'PY'
from pathlib import Path
import re

root = Path('.')

required_files = [
    'docs/product/PRODUCT_OVERVIEW.md',
    'docs/product/CAPABILITY_MATRIX.md',
    'docs/product/FEATURE_MATURITY.md',
    'docs/product/EDITION_PACKAGING.md',
    'docs/product/DIFFERENTIATION_AND_MOAT.md',
    'docs/release/RELEASE_CRITERIA.md',
    'docs/release/UPGRADE_AND_MIGRATION.md',
    'docs/release/BACKUP_AND_RESTORE.md',
    'docs/release/SUPPORT_RUNBOOK.md',
    'docs/release/SECURITY_MODEL.md',
    'docs/getting-started/QUICKSTART.md',
    'docs/getting-started/FIRST_INCIDENT_GUIDE.md',
    'docs/internal/private/PRICING_STRATEGY.md',
    'docs/internal/private/RISK_REGISTER.md',
]
missing = [p for p in required_files if not (root / p).exists()]
if missing:
    raise SystemExit(f"missing required product-system docs: {missing}")

feature = (root / 'docs/product/FEATURE_MATURITY.md').read_text(encoding='utf-8')
for label in ['GA', 'Beta', 'Experimental', 'Unsupported', 'Roadmap']:
    if f'### {label}' not in feature and label != 'Roadmap':
        raise SystemExit(f"feature maturity missing section: {label}")
if 'Roadmap' not in feature:
    raise SystemExit('feature maturity missing roadmap language')

transport_consistency_targets = [
    'docs/product/CAPABILITY_MATRIX.md',
    'docs/product/FEATURE_MATURITY.md',
    'docs/release/KNOWN_LIMITATIONS.md',
]
for rel in transport_consistency_targets:
    text = (root / rel).read_text(encoding='utf-8')
    for token in ['BLE', 'HTTP']:
        if token not in text or 'unsupported' not in text.lower():
            raise SystemExit(f"{rel} must keep explicit unsupported posture for {token}")

scan_dirs = [
    root / 'docs/product',
    root / 'docs/release',
    root / 'docs/getting-started',
    root / 'docs/internal/private',
]
link_re = re.compile(r'\[[^\]]+\]\(([^)]+)\)')
errors = []
for d in scan_dirs:
    for md in d.rglob('*.md'):
        txt = md.read_text(encoding='utf-8')
        for m in link_re.finditer(txt):
            href = m.group(1).strip()
            if href.startswith(('http://', 'https://', '#', 'mailto:')):
                continue
            path = href.split('#', 1)[0]
            if not path:
                continue
            target = (md.parent / path).resolve()
            if not target.exists():
                errors.append(f"{md.relative_to(root)} -> {href}")
if errors:
    joined = "\n".join(errors[:40])
    raise SystemExit(f"broken links in product-system docs:\n{joined}")

print('product-system verification passed')
PY
