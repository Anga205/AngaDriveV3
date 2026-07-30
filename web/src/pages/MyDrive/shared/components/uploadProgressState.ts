export interface UploadProgressSnapshot {
    totalOriginalBytes: number;
    totalCompressedBytes: number;
    acknowledgedCompressedBytes: number;
    totalChunksGenerated: number;
    totalChunksAcknowledged: number;
    compressionFinished: boolean;
    serverCompleted: boolean;
}

const clamp = (value: number, min: number, max: number): number => Math.min(max, Math.max(min, value));

export function computeAckDrivenProgress(snapshot: UploadProgressSnapshot): number {
    if (snapshot.serverCompleted) return 100;
    if (snapshot.totalOriginalBytes <= 0) return 0;

    let candidate = 0;
    if (snapshot.compressionFinished && snapshot.totalCompressedBytes > 0) {
        candidate = Math.floor((snapshot.acknowledgedCompressedBytes / snapshot.totalCompressedBytes) * 99);
        if (snapshot.totalChunksGenerated > 0 && snapshot.totalChunksAcknowledged >= snapshot.totalChunksGenerated) {
            candidate = Math.max(candidate, 99);
        }
    } else {
        // While compression is in-flight, use a conservative denominator tied to full file size.
        candidate = Math.floor((snapshot.acknowledgedCompressedBytes / snapshot.totalOriginalBytes) * 95);
    }

    return clamp(candidate, 0, 99);
}

export function canSendFinalize(snapshot: UploadProgressSnapshot): boolean {
    if (!snapshot.compressionFinished) return false;
    if (snapshot.totalChunksGenerated <= 0) return false;
    return snapshot.totalChunksAcknowledged >= snapshot.totalChunksGenerated;
}

export class UploadProgressTracker {
    private snapshot: UploadProgressSnapshot;
    private progress = 0;

    constructor(totalOriginalBytes: number) {
        this.snapshot = {
            totalOriginalBytes,
            totalCompressedBytes: 0,
            acknowledgedCompressedBytes: 0,
            totalChunksGenerated: 0,
            totalChunksAcknowledged: 0,
            compressionFinished: false,
            serverCompleted: false,
        };
    }

    recordChunkGenerated(chunkBytes: number): number {
        this.snapshot.totalChunksGenerated += 1;
        this.snapshot.totalCompressedBytes += Math.max(0, chunkBytes);
        return this.recalculate();
    }

    recordChunkAcknowledged(ackBytes: number): number {
        this.snapshot.totalChunksAcknowledged += 1;
        this.snapshot.acknowledgedCompressedBytes += Math.max(0, ackBytes);
        return this.recalculate();
    }

    markCompressionFinished(): number {
        this.snapshot.compressionFinished = true;
        return this.recalculate();
    }

    markServerCompleted(): number {
        this.snapshot.serverCompleted = true;
        this.progress = 100;
        return this.progress;
    }

    getProgress(): number {
        return this.progress;
    }

    getSnapshot(): UploadProgressSnapshot {
        return { ...this.snapshot };
    }

    private recalculate(): number {
        const next = computeAckDrivenProgress(this.snapshot);
        if (next > this.progress) {
            this.progress = next;
        }
        return this.progress;
    }
}