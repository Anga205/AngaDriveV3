import { describe, expect, it } from 'vitest';
import { UploadProgressTracker, canSendFinalize } from './uploadProgressState';

describe('upload progress state machine', () => {
    it('keeps progress below 100 when compression finishes with unacknowledged chunks', () => {
        const tracker = new UploadProgressTracker(10_000);
        tracker.recordChunkGenerated(2000);
        tracker.recordChunkGenerated(2000);
        tracker.recordChunkGenerated(2000);
        tracker.recordChunkAcknowledged(2000);
        tracker.markCompressionFinished();

        expect(tracker.getProgress()).toBeLessThan(100);
    });

    it('keeps progress below 100 when all chunks are generated but not all acknowledged', () => {
        const tracker = new UploadProgressTracker(10_000);
        tracker.recordChunkGenerated(2500);
        tracker.recordChunkGenerated(2500);
        tracker.recordChunkGenerated(2500);
        tracker.recordChunkGenerated(2500);
        tracker.recordChunkAcknowledged(2500);
        tracker.recordChunkAcknowledged(2500);
        tracker.markCompressionFinished();

        expect(tracker.getProgress()).toBeLessThan(100);
    });

    it('keeps progress below 100 when chunks are sent but ACKs are missing', () => {
        const tracker = new UploadProgressTracker(10_000);
        tracker.recordChunkGenerated(1000);
        tracker.recordChunkGenerated(1000);
        tracker.recordChunkGenerated(1000);
        tracker.recordChunkGenerated(1000);
        tracker.markCompressionFinished();

        expect(tracker.getProgress()).toBeLessThan(100);
        expect(tracker.getProgress()).toBe(0);
    });

    it('keeps progress below 100 when all chunks are ACKed but FILE_COMPLETE is missing', () => {
        const tracker = new UploadProgressTracker(10_000);
        tracker.recordChunkGenerated(1500);
        tracker.recordChunkGenerated(1500);
        tracker.recordChunkGenerated(1500);
        tracker.recordChunkAcknowledged(1500);
        tracker.recordChunkAcknowledged(1500);
        tracker.recordChunkAcknowledged(1500);
        tracker.markCompressionFinished();

        expect(tracker.getProgress()).toBe(99);
        expect(tracker.getProgress()).toBeLessThan(100);
    });

    it('reaches 100 only after FILE_COMPLETE state arrives', () => {
        const tracker = new UploadProgressTracker(10_000);
        tracker.recordChunkGenerated(1000);
        tracker.recordChunkAcknowledged(1000);
        tracker.markCompressionFinished();

        expect(tracker.getProgress()).toBeLessThan(100);
        tracker.markServerCompleted();
        expect(tracker.getProgress()).toBe(100);
    });

    it('never reaches 100 without FILE_COMPLETE even after all ACKed work', () => {
        const tracker = new UploadProgressTracker(10_000);
        tracker.recordChunkGenerated(3000);
        tracker.recordChunkGenerated(3000);
        tracker.recordChunkAcknowledged(3000);
        tracker.recordChunkAcknowledged(3000);
        tracker.markCompressionFinished();

        expect(tracker.getProgress()).toBe(99);
        expect(tracker.getProgress()).not.toBe(100);
    });

    it('does not allow FINALIZE before the final chunk ACK', () => {
        const tracker = new UploadProgressTracker(10_000);
        tracker.recordChunkGenerated(1000);
        tracker.recordChunkGenerated(1000);
        tracker.recordChunkAcknowledged(1000);
        tracker.markCompressionFinished();

        expect(canSendFinalize(tracker.getSnapshot())).toBe(false);

        tracker.recordChunkAcknowledged(1000);
        expect(canSendFinalize(tracker.getSnapshot())).toBe(true);
    });

    it('maintains monotonic progress values', () => {
        const tracker = new UploadProgressTracker(10_000);
        const seen: number[] = [];

        tracker.recordChunkGenerated(2000);
        seen.push(tracker.getProgress());
        tracker.recordChunkAcknowledged(500);
        seen.push(tracker.getProgress());
        tracker.recordChunkGenerated(5000);
        seen.push(tracker.getProgress());
        tracker.recordChunkAcknowledged(2000);
        seen.push(tracker.getProgress());
        tracker.markCompressionFinished();
        seen.push(tracker.getProgress());
        tracker.recordChunkAcknowledged(1000);
        seen.push(tracker.getProgress());

        for (let i = 1; i < seen.length; i += 1) {
            expect(seen[i]).toBeGreaterThanOrEqual(seen[i - 1]);
        }
    });

    it('never exceeds 99 before FILE_COMPLETE', () => {
        const tracker = new UploadProgressTracker(1);
        tracker.recordChunkGenerated(1000);
        tracker.recordChunkAcknowledged(1000);
        tracker.markCompressionFinished();

        expect(tracker.getProgress()).toBeLessThanOrEqual(99);
        tracker.markServerCompleted();
        expect(tracker.getProgress()).toBe(100);
    });

    it('maintains independent progress per file tracker', () => {
        const fileA = new UploadProgressTracker(10_000);
        const fileB = new UploadProgressTracker(10_000);

        fileA.recordChunkGenerated(2000);
        fileA.recordChunkAcknowledged(2000);

        fileB.recordChunkGenerated(2000);
        fileB.recordChunkAcknowledged(500);

        expect(fileA.getProgress()).toBeGreaterThan(fileB.getProgress());
        expect(fileA.getProgress()).toBeLessThan(100);
        expect(fileB.getProgress()).toBeLessThan(100);
    });
});