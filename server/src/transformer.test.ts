import { describe, it, expect } from 'vitest';
import { transformToPrometheus } from './transformer.js';

describe('transformToPrometheus', () => {
  it('should transform a single metric item', () => {
    const payload = [
      {
        timestamp: 1000,
        metrics: { cpu_usage_percent: 42.5, hostname: 'server-01' },
      },
    ];

    const result = transformToPrometheus(payload as any, 1);

    expect(result).toContain('veona_cpu_usage_percent');
    expect(result).toContain('probe_id="1"');
    expect(result).toContain('hostname="server-01"');
    expect(result).toContain('42.5');
    expect(result).toContain('1000000'); // timestamp * 1000
  });

  it('should handle multiple metric items', () => {
    const payload = [
      {
        timestamp: 1000,
        metrics: { cpu_usage_percent: 42.5, hostname: 'server-01' },
      },
      {
        timestamp: 1001,
        metrics: { mem_used_percent: 65.0, hostname: 'server-01' },
      },
    ];

    const result = transformToPrometheus(payload as any, 'probe-abc');

    expect(result).toContain('veona_cpu_usage_percent');
    expect(result).toContain('veona_mem_used_percent');
    expect(result).toContain('probe_id="probe-abc"');
  });

  it('should skip non-numeric values', () => {
    const payload = [
      {
        timestamp: 1000,
        metrics: {
          cpu_usage_percent: 50,
          hostname: 'server-01',
          status: 'online',
          enabled: true,
        },
      },
    ];

    const result = transformToPrometheus(payload as any, 1);

    expect(result).toContain('veona_cpu_usage_percent');
    expect(result).not.toContain('veona_status');
    expect(result).not.toContain('veona_enabled');
  });

  it('should clean metric keys', () => {
    const payload = [
      {
        timestamp: 1000,
        metrics: { 'cpu.usage%': 90, hostname: 'server-01' },
      },
    ];

    const result = transformToPrometheus(payload as any, 1);

    expect(result).toContain('veona_cpu_usage_');
  });

  it('should use unknown_host when hostname is missing', () => {
    const payload = [
      {
        timestamp: 1000,
        metrics: { cpu_usage_percent: 50 },
      },
    ];

    const result = transformToPrometheus(payload as any, 1);

    expect(result).toContain('hostname="unknown_host"');
  });

  it('should handle empty payload', () => {
    const result = transformToPrometheus([], '1');
    expect(result).toBe('');
  });

  it('should handle items with no metrics', () => {
    const payload = [
      {
        timestamp: 1000,
        metrics: {},
      },
    ];

    const result = transformToPrometheus(payload as any, 1);
    expect(result).toBe('');
  });
});
