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
        expect(tracker.isDataReceived()).toBe(false);
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
        expect(tracker.isDataReceived()).toBe(false);
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
        expect(tracker.isDataReceived()).toBe(false);
    });

    it('reaches 100 as soon as all chunks are generated and ACKed (DATA_RECEIVED)', () => {
        const tracker = new UploadProgressTracker(10_000);
        tracker.recordChunkGenerated(1500);
        tracker.recordChunkGenerated(1500);
        tracker.recordChunkGenerated(1500);
        tracker.recordChunkAcknowledged(1500);
        tracker.recordChunkAcknowledged(1500);
        tracker.recordChunkAcknowledged(1500);
        tracker.markCompressionFinished();

        expect(tracker.getProgress()).toBe(100);
        expect(tracker.isDataReceived()).toBe(true);
    });

    it('marks data received explicitly via markDataReceived', () => {
        const tracker = new UploadProgressTracker(10_000);
        tracker.recordChunkGenerated(1000);
        tracker.recordChunkAcknowledged(1000);
        tracker.markCompressionFinished();

        tracker.markDataReceived();
        expect(tracker.getProgress()).toBe(100);
        expect(tracker.isDataReceived()).toBe(true);
    });

    it('reaches 100 when serverCompleted is marked', () => {
        const tracker = new UploadProgressTracker(10_000);
        tracker.recordChunkGenerated(1000);
        tracker.recordChunkAcknowledged(500);

        expect(tracker.getProgress()).toBeLessThan(100);
        tracker.markServerCompleted();
        expect(tracker.getProgress()).toBe(100);
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