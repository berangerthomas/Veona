export interface MetricItem {
  timestamp: number;
  metrics: Record<string, string | number | boolean>;
}

export function transformToPrometheus(payload: MetricItem[], probeId: string | number): string {
  let promText = '';

  for (const item of payload) {
    if (!item.metrics) continue;

    const timestampMs = item.timestamp * 1000;
    const hostname = (item.metrics.hostname as string) || 'unknown_host';

    for (const [key, value] of Object.entries(item.metrics)) {
      if (typeof value === 'number') {
        // Metric names should be cleaned (replace dots/dashes if any)
        const cleanKey = key.replace(/[^a-zA-Z0-9_]/g, '_');
        promText += `veona_${cleanKey}{probe_id="${probeId}", hostname="${hostname}"} ${value} ${timestampMs}\n`;
      }
    }
  }

  return promText;
}
