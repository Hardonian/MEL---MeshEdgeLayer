# MEL Revenue Model

**Date:** 2026-04-10  
**Status:** Active

---

## Node-Based Licensing

### Tiers

| Tier | Price | Nodes | Storage | Support |
|------|-------|-------|--------|---------|
| Community | $0 | 5 | 1GB | Community |
| Professional | $199/node/yr | Unlimited | 100GB | Email (48h) |
| Enterprise | Custom | Unlimited | Unlimited | Dedicated CSM |

---

## License System

### 1. License Generation

```typescript
// Generate license key
async function createLicense(orgId: string, tier: string, nodes: number) {
  const license = {
    key: generateSecureKey(), // crypto.randomUUID()
    orgId,
    tier,
    nodes,
    created: new Date(),
    expires: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year
    features: TIER_FEATURES[tier],
  };
  
  await db.licenses.create({ data: license });
  return license.key;
}
```

### 2. License Validation

```typescript
// Validate on node startup
async function validateLicense(key: string): Promise<boolean> {
  const license = await db.licenses.findUnique({ where: { key }});
  
  if (!license) return false;
  if (license.expires < new Date()) return false;
  if (license.nodes < currentNodeCount()) return false;
  
  // Update last validated
  await db.licenses.update({
    where: { key },
    data: { lastValidated: new Date() },
  });
  
  return true;
}
```

### 3. Usage Reporting

```typescript
// Track node usage
async function reportNodeUsage(licenseKey: string) {
  const license = await db.licenses.findUnique({ where: { key: licenseKey }});
  const nodes = await db.nodes.count({ where: { licenseKey }});
  
  await db.usageRecords.create({
    data: {
      licenseId: license.id,
      nodeCount: nodes,
      reported: new Date(),
    },
  });
  
  // Alert on overuse
  if (nodes > license.nodes) {
    await sendAlert(license.orgId, 'Node limit exceeded');
  }
}
```

---

## Revenue Tracking

```typescript
// Annual recurring revenue
async function calculateARR() {
  const licenses = await db.licenses.findMany({
    where: { 
      tier: { not: 'community' },
      expires: { gt: new Date() },
    },
  });
  
  let arr = 0;
  for (const license of licenses) {
    if (license.tier === 'professional') {
      arr += license.nodes * 199;
    } else {
      arr += ENTERPRISE_PRICE; // Custom
    }
  }
  
  return arr;
}
```

---

## Environment Variables

```
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
LICENSE_KEY_SECRET=your-secret-key...
```

---

*Status: Ready for production*