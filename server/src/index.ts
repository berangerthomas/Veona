import { serve } from '@hono/node-server';
import { Hono } from 'hono';
import Database from 'better-sqlite3';
import { drizzle } from 'drizzle-orm/better-sqlite3';
import { eq } from 'drizzle-orm';
import * as schema from './db/schema.js';
import zlib from 'zlib';
import util from 'util';
import { z } from 'zod';
import { LRUCache } from 'lru-cache';
import { transformToPrometheus } from './transformer.js';
import fs from 'fs';

const unzip = util.promisify(zlib.gunzip);

type Env = {
  Variables: {
    probeId: string | number;
  }
};

const app = new Hono<Env>();

// Zod Schemas for Validation
const MetricItemSchema = z.object({
  timestamp: z.number(),
  metrics: z.record(z.union([z.string(), z.number(), z.boolean()])),
});

const IngestionPayloadSchema = z.array(MetricItemSchema);

if (!fs.existsSync('data')) {
    fs.mkdirSync('data');
}

// 1. Initialize SQLite & Drizzle (Control Plane)
const sqlite = new Database('data/veona.db');
const db = drizzle(sqlite, { schema });

// 1.1. Create tables if not exist (simple SQLite bootstrap, no Drizzle migrations needed)
sqlite.exec(`
  CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at INTEGER
  );
  CREATE TABLE IF NOT EXISTS probes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    api_key TEXT NOT NULL UNIQUE,
    status TEXT DEFAULT 'active' NOT NULL,
    user_id INTEGER REFERENCES users(id),
    last_seen_at INTEGER,
    created_at INTEGER
  );
  CREATE UNIQUE INDEX IF NOT EXISTS idx_probes_api_key ON probes(api_key);
  CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);
`);

// 1.5. In-Memory Cache for API Keys (LRU with 10min TTL)
const apiKeyCache = new LRUCache<string, number>({
  max: 500,
  ttl: 1000 * 60 * 10, // 10 minutes
});

// 1.6. Simple Rate Limiter Cache
const rateLimitCache = new LRUCache<string, number>({
  max: 1000,
  ttl: 60 * 1000, // 1 minute window
});

// Configuration
// We use the prometheus import endpoint for VictoriaMetrics
const VICTORIA_METRICS_URL = process.env.VICTORIA_METRICS_URL || 'http://localhost:8428/api/v1/import/prometheus';

// 2. Middleware: Rate Limiting
const rateLimiter = async (c: any, next: any) => {
  const ip = c.req.header('x-forwarded-for') || 'anonymous';
  const count = rateLimitCache.get(ip) || 0;
  if (count >= 300) { // 300 requests per minute per IP
    return c.json({ error: 'Rate limit exceeded' }, 429);
  }
  rateLimitCache.set(ip, count + 1);
  await next();
};

// 2.5. Middleware: Auth Validation
const validateToken = async (c: any, next: any) => {
  const authHeader = c.req.header('Authorization');
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return c.json({ error: 'Missing or invalid token' }, 401);
  }

  const token = authHeader.split(' ')[1];

  // Check LRU Cache first
  let probeId = apiKeyCache.get(token);

  if (probeId === undefined) {
    // Fallback to SQLite DB
    const probe = db.select({ id: schema.probes.id }).from(schema.probes).where(eq(schema.probes.apiKey, token)).get();
    if (!probe) {
      return c.json({ error: 'Unauthorized' }, 401);
    }
    probeId = probe.id;
    // Store in cache
    apiKeyCache.set(token, probeId);
  }
  
  // Attach probeId to context for later use
  c.set('probeId', probeId);
  await next();
};

// 3. Health Check
app.get('/health', (c) => c.text('Veona API OK'));

// 4. Ingestion Route (Data Plane)
app.post('/api/metrics', rateLimiter, validateToken, async (c) => {
  try {
    // Read raw body to handle gzip
    const bodyBuffer = await c.req.arrayBuffer();
    const isGzip = c.req.header('Content-Encoding') === 'gzip';
    
    let rawJson = '';
    if (isGzip) {
        const unzipped = await unzip(bodyBuffer);
        rawJson = unzipped.toString('utf-8');
    } else {
        const decoder = new TextDecoder('utf-8');
        rawJson = decoder.decode(bodyBuffer);
    }
    
    const json = JSON.parse(rawJson);
    const result = IngestionPayloadSchema.safeParse(json);

    if (!result.success) {
      return c.json({ error: 'Invalid payload schema', details: result.error.format() }, 400);
    }

    const payload = result.data;
    
    // Retrieve probeId from context (set by validateToken middleware)
    const probeId = c.get('probeId') ?? 'unknown';

    // Update lastSeenAt in background (non-blocking for ingestion)
    if (typeof probeId === 'number') {
        db.update(schema.probes)
            .set({ lastSeenAt: new Date() })
            .where(eq(schema.probes.id, probeId))
            .run();
    }

    // Transform logic: Array of JSON -> Prometheus Text Format
    const promText = transformToPrometheus(payload as any, probeId);

    // Send formatted Prometheus metrics to VictoriaMetrics
    const vmResponse = await fetch(VICTORIA_METRICS_URL, {
        method: 'POST',
        headers: { 'Content-Type': 'text/plain' },
        body: promText,
    });

    if (!vmResponse.ok) {
        console.error('Failed to forward to VictoriaMetrics:', vmResponse.statusText);
        // Propagate status from VictoriaMetrics if applicable, or 500
        const status = vmResponse.status >= 500 ? 500 : vmResponse.status;
        return c.json({ error: 'Data Plane Error', details: vmResponse.statusText }, status as any);
    }

    return c.json({ success: true, message: 'Metrics ingested' });
  } catch (e: any) {
    console.error('Ingestion error:', e);
    if (e instanceof SyntaxError) {
        return c.json({ error: 'Invalid JSON' }, 400);
    }
    return c.json({ error: 'Internal Server Error' }, 500);
  }
});

// Start Server
const port = 3000;
console.log(`Starting Veona Server on port ${port}...`);

serve({
  fetch: app.fetch,
  port
});
