import { describe, it, expect } from 'vitest';

/**
 * CPU-side upload preparation benchmark.
 *
 * This measures the client-side cost of chunk preparation for both upload
 * modes, isolating the CPU work (compression vs. raw slicing) from the
 * network transfer, which cannot be reproduced in a unit test.
 *
 * The hypothesis under test:
 *   On some mobile/ARM devices, gzip CPU cost > network bandwidth savings.
 *
 * This benchmark only measures the CPU side. It does NOT measure end-to-end
 * upload time (which requires a real network). The results distinguish:
 *   - compression CPU time (gzip mode)
 *   - chunk preparation time (both modes)
 *   - total bytes produced (compressed vs. raw)
 *   - upload preparation throughput (bytes/sec of CPU work)
 */

const CHUNK_SIZE = 7 * 1024 * 1024; // 7MB, matches production

// A ~14MB pseudo-random-ish buffer (incompressible-ish, like real binary data)
function makeTestBuffer(size: number): Uint8Array {
  const buf = new Uint8Array(size);
  for (let i = 0; i < size; i++) {
    // Simple deterministic pattern with enough entropy to be realistic
    buf[i] = (i * 31 + 7) % 256;
  }
  return buf;
}

async function gzipChunk(chunk: Uint8Array): Promise<Blob> {
  // Use a ReadableStream directly from the buffer to avoid Blob.stream()
  // which is not available in jsdom.
  const rs = new ReadableStream({
    start(controller) {
      controller.enqueue(chunk);
      controller.close();
    },
  });
  const compressed = rs.pipeThrough(new CompressionStream('gzip'));
  return new Response(compressed).blob();
}

describe('upload preparation benchmark (CPU-side only)', () => {
  it('measures gzip vs raw chunk preparation', async () => {
    const fileSize = 14 * 1024 * 1024; // 14 MB test file
    const buffer = makeTestBuffer(fileSize);
    const blob = new Blob([buffer.buffer as ArrayBuffer]);
    const totalChunks = Math.ceil(fileSize / CHUNK_SIZE);

    // --- Desktop path: slice + gzip each chunk ---
    const gzipStart = performance.now();
    let compressedTotal = 0;
    for (let i = 0; i < totalChunks; i++) {
      const start = i * CHUNK_SIZE;
      const end = Math.min(start + CHUNK_SIZE, fileSize);
      const chunkBlob = blob.slice(start, end);
      const compressed = await gzipChunk(new Uint8Array(await chunkBlob.arrayBuffer()));
      compressedTotal += compressed.size;
    }
    const gzipElapsed = performance.now() - gzipStart;

    // --- Mobile path: slice only (no compression) ---
    const rawStart = performance.now();
    let rawTotal = 0;
    for (let i = 0; i < totalChunks; i++) {
      const start = i * CHUNK_SIZE;
      const end = Math.min(start + CHUNK_SIZE, fileSize);
      const chunkBlob = blob.slice(start, end);
      rawTotal += chunkBlob.size;
    }
    const rawElapsed = performance.now() - rawStart;

    // --- Report ---
    // eslint-disable-next-line no-console
    console.log('\n=== Upload Preparation Benchmark (CPU-side) ===');
    // eslint-disable-next-line no-console
    console.log(`File size: ${fileSize} bytes, ${totalChunks} chunks of ${CHUNK_SIZE} bytes`);
    // eslint-disable-next-line no-console
    console.log(`[gzip]  total bytes produced: ${compressedTotal} (${(compressedTotal / fileSize * 100).toFixed(1)}% of original)`);
    // eslint-disable-next-line no-console
    console.log(`[raw]   total bytes produced: ${rawTotal} (100% of original)`);
    // eslint-disable-next-line no-console
    console.log(`[gzip]  chunk prep time: ${gzipElapsed.toFixed(1)}ms, throughput: ${(fileSize / gzipElapsed * 1000 / 1024 / 1024).toFixed(2)} MB/s`);
    // eslint-disable-next-line no-console
    console.log(`[raw]   chunk prep time: ${rawElapsed.toFixed(1)}ms, throughput: ${(fileSize / rawElapsed * 1000 / 1024 / 1024).toFixed(2)} MB/s`);
    // eslint-disable-next-line no-console
    console.log(`[CPU]   gzip overhead: ${(gzipElapsed / rawElapsed).toFixed(2)}x slower than raw`);
    // eslint-disable-next-line no-console
    console.log('==============================================\n');

    // Sanity assertions: both modes produce the expected chunk counts
    expect(totalChunks).toBe(2);
    // Raw must produce exactly the original size
    expect(rawTotal).toBe(fileSize);
    // gzip must produce SOME output (even if incompressible it won't be zero)
    expect(compressedTotal).toBeGreaterThan(0);
  }, 60000);
});