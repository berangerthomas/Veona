import { sqliteTable, text, integer } from 'drizzle-orm/sqlite-core';

// Control Plane: Users (Admin Dashboard)
export const users = sqliteTable('users', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  email: text('email').notNull().unique(),
  passwordHash: text('password_hash').notNull(),
  createdAt: integer('created_at', { mode: 'timestamp' }).$defaultFn(() => new Date()),
});

// Control Plane: Probes (Agents)
export const probes = sqliteTable('probes', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  name: text('name').notNull(),
  apiKey: text('api_key').notNull().unique(),
  status: text('status', { enum: ['active', 'offline'] }).default('active').notNull(),
  userId: integer('user_id').references(() => users.id),
  lastSeenAt: integer('last_seen_at', { mode: 'timestamp' }),
  createdAt: integer('created_at', { mode: 'timestamp' }).$defaultFn(() => new Date()),
});
