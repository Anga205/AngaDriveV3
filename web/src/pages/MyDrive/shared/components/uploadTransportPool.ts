import type { AuthDetails, SelectableFile } from "../types";
import { UploadProgressTracker, canSendFinalize } from "./uploadProgressState";

const CHUNK_SIZE = 7 * 1024 * 1024; // 7MB
export const UPLOAD_WEBSOCKET_COUNT = 3;

export interface UploadQueueItem {
    queueItemId: string;
    uploadId: string;
    fileId: string;
    fileName: string;
    fileSize: number;
    chunkIndex: number;
    encoding: string;
    payload: Uint8Array;
    auth: AuthDetails;
    collectionId?: string;
    retries: number;
    inFlight: boolean;
}

export interface ActiveFileUploadSession {
    selectableFile: SelectableFile;
    uploadId: string;
    auth: AuthDetails;
    collectionId?: string;
    tracker: UploadProgressTracker;
    updateProgress: (progress: number) => void;
    onDataReceived?: () => void;
    onServerCompleted?: () => void;
    onError?: (error: Error) => void;
    waitWhilePaused?: () => Promise<void>;
    isPaused?: () => boolean;
    shouldCancel?: () => boolean;
    cancelled?: boolean;
    completed?: boolean;
    dataReceivedNotified?: boolean;
    totalChunksExpected?: number;
    compressionFinished?: boolean;
}

class UploadWorkerConnection {
    id: number;
    private ws: WebSocket | null = null;
    private connectingPromise: Promise<void> | null = null;
    private isClosedExplicitly = false;
    private onAckCallback: (ack: { uploadId: string; fileId?: string; chunkIndex: number; payloadBytes: number }) => void;
    private onDataReceivedCallback: (payload: any) => void;
    private onFileCompleteCallback: (payload: any) => void;
    private onErrorCallback: (error: Error, uploadId?: string, fileId?: string) => void;
    private onSocketLostCallback: (workerId: number) => void;

    constructor(
        id: number,
        callbacks: {
            onAck: (ack: { uploadId: string; fileId?: string; chunkIndex: number; payloadBytes: number }) => void;
            onDataReceived: (payload: any) => void;
            onFileComplete: (payload: any) => void;
            onError: (error: Error, uploadId?: string, fileId?: string) => void;
            onSocketLost: (workerId: number) => void;
        }
    ) {
        this.id = id;
        this.onAckCallback = callbacks.onAck;
        this.onDataReceivedCallback = callbacks.onDataReceived;
        this.onFileCompleteCallback = callbacks.onFileComplete;
        this.onErrorCallback = callbacks.onError;
        this.onSocketLostCallback = callbacks.onSocketLost;
    }

    async ensureOpen(): Promise<WebSocket> {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            return this.ws;
        }
        if (this.connectingPromise) {
            await this.connectingPromise;
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                return this.ws;
            }
        }

        this.connectingPromise = new Promise<void>((resolve, reject) => {
            const baseUrl = import.meta.env.DEV ? 'http://localhost:8080' : window.location.origin;
            const wsUrl = `${baseUrl.replace(/^http/, 'ws')}/upload/ws/pool`;

            const socket = new WebSocket(wsUrl);
            socket.binaryType = 'arraybuffer';

            socket.onopen = () => {
                this.ws = socket;
                this.connectingPromise = null;
                resolve();
            };

            socket.onmessage = (event) => {
                try {
                    const payload = typeof event.data === 'string' ? JSON.parse(event.data) : null;
                    if (!payload) return;

                    if (payload.type === 'CHUNK_ACK') {
                        this.onAckCallback({
                            uploadId: payload.uploadId,
                            fileId: payload.fileId,
                            chunkIndex: Number(payload.chunkIndex ?? -1),
                            payloadBytes: Number(payload.payloadBytes || 0),
                        });
                        return;
                    }

                    if (payload.type === 'DATA_RECEIVED') {
                        this.onDataReceivedCallback(payload);
                        return;
                    }

                    if (payload.type === 'FILE_COMPLETE') {
                        this.onFileCompleteCallback(payload);
                        return;
                    }

                    if (payload.type === 'UPLOAD_ERROR') {
                        const err = new Error(payload.message || 'Upload error');
                        this.onErrorCallback(err, payload.uploadId, payload.fileId);
                        return;
                    }
                } catch (err: any) {
                    console.error('Error handling websocket message:', err);
                }
            };

            socket.onerror = () => {
                const err = new Error(`WebSocket worker ${this.id} encountered an error`);
                if (this.connectingPromise) {
                    this.connectingPromise = null;
                    reject(err);
                }
                this.onErrorCallback(err);
            };

            socket.onclose = () => {
                this.ws = null;
                this.connectingPromise = null;
                if (!this.isClosedExplicitly) {
                    this.onSocketLostCallback(this.id);
                }
            };
        });

        await this.connectingPromise;
        return this.ws as WebSocket;
    }

    async sendFrame(frameBuffer: ArrayBuffer): Promise<void> {
        const socket = await this.ensureOpen();
        if (socket.readyState !== WebSocket.OPEN) {
            throw new Error(`Socket worker ${this.id} is not open`);
        }
        socket.send(frameBuffer);
    }

    sendControl(control: Record<string, any>): void {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(control));
        }
    }

    close(): void {
        this.isClosedExplicitly = true;
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }
}

export class GlobalUploadTransportManager {
    private static instance: GlobalUploadTransportManager | null = null;
    private workers: UploadWorkerConnection[] = [];
    private workerCount: number;
    private queue: UploadQueueItem[] = [];
    private inFlightMap = new Map<string, UploadQueueItem>(); // key: `${uploadId}:${chunkIndex}`
    private sessions = new Map<string, ActiveFileUploadSession>(); // key: uploadId
    private isPumping = false;
    private isPaused = false;

    private constructor(workerCount: number = UPLOAD_WEBSOCKET_COUNT) {
        this.workerCount = workerCount;
        this.initWorkers();
    }

    static getInstance(workerCount: number = UPLOAD_WEBSOCKET_COUNT): GlobalUploadTransportManager {
        if (!GlobalUploadTransportManager.instance) {
            GlobalUploadTransportManager.instance = new GlobalUploadTransportManager(workerCount);
        }
        return GlobalUploadTransportManager.instance;
    }

    private initWorkers() {
        this.workers = [];
        for (let i = 0; i < this.workerCount; i++) {
            const worker = new UploadWorkerConnection(i, {
                onAck: (ack) => this.handleAck(ack),
                onDataReceived: (payload) => this.handleDataReceived(payload),
                onFileComplete: (payload) => this.handleFileComplete(payload),
                onError: (error, uploadId, fileId) => this.handleError(error, uploadId, fileId),
                onSocketLost: (workerId) => this.handleSocketLost(workerId),
            });
            this.workers.push(worker);
        }
    }

    setPaused(paused: boolean) {
        this.isPaused = paused;
        if (!paused) {
            this.pump();
        }
    }

    registerSession(session: ActiveFileUploadSession) {
        this.sessions.set(session.uploadId, session);
        this.startCompressor(session);
    }

    unregisterSession(uploadId: string) {
        this.sessions.delete(uploadId);
        // Remove pending items for this uploadId from queue
        this.queue = this.queue.filter(item => item.uploadId !== uploadId);
        // Remove in-flight
        Array.from(this.inFlightMap.keys()).forEach(key => {
            if (key.startsWith(`${uploadId}:`)) {
                this.inFlightMap.delete(key);
            }
        });
    }

    private async startCompressor(session: ActiveFileUploadSession) {
        const file = session.selectableFile.file;
        const uploadId = session.uploadId;
        const fileId = session.selectableFile.uniqueId;
        const encoding = 'gzip-stream-v1';

        try {
            const reader = file.stream().pipeThrough(new CompressionStream('gzip')).getReader();
            let buffer = new Uint8Array(0);
            let chunkIndex = 0;

            while (true) {
                if (session.shouldCancel?.() || session.cancelled) return;
                if (session.waitWhilePaused && session.isPaused && session.isPaused()) {
                    await session.waitWhilePaused();
                }

                // Backpressure: wait if queue has too many items for this upload session
                while (this.getQueueCountForUpload(uploadId) >= 4) {
                    if (session.shouldCancel?.() || session.cancelled) return;
                    await new Promise(r => setTimeout(r, 50));
                }

                const result = await reader.read();
                if (result.done) break;

                const next = new Uint8Array(buffer.length + result.value.length);
                next.set(buffer, 0);
                next.set(result.value, buffer.length);
                buffer = next;

                while (buffer.length >= CHUNK_SIZE) {
                    const slice = buffer.slice(0, CHUNK_SIZE);
                    buffer = buffer.slice(CHUNK_SIZE);

                    session.tracker.recordChunkGenerated(slice.length);
                    this.enqueueChunk({
                        queueItemId: `${uploadId}:${chunkIndex}`,
                        uploadId,
                        fileId,
                        fileName: file.name,
                        fileSize: file.size,
                        chunkIndex,
                        encoding,
                        payload: slice,
                        auth: session.auth,
                        collectionId: session.collectionId,
                        retries: 0,
                        inFlight: false,
                    });
                    chunkIndex++;
                }
            }

            if (buffer.length > 0) {
                session.tracker.recordChunkGenerated(buffer.length);
                this.enqueueChunk({
                    queueItemId: `${uploadId}:${chunkIndex}`,
                    uploadId,
                    fileId,
                    fileName: file.name,
                    fileSize: file.size,
                    chunkIndex,
                    encoding,
                    payload: buffer,
                    auth: session.auth,
                    collectionId: session.collectionId,
                    retries: 0,
                    inFlight: false,
                });
                chunkIndex++;
            }

            session.compressionFinished = true;
            session.totalChunksExpected = chunkIndex;
            session.tracker.markCompressionFinished();
            this.emitSessionProgress(session, 'COMPRESSION_FINISHED');
            this.checkSessionCompletion(session);
        } catch (err: any) {
            console.error(`Compression error for ${file.name}:`, err);
            session.onError?.(err);
        }
    }

    private getQueueCountForUpload(uploadId: string): number {
        return this.queue.filter(item => item.uploadId === uploadId).length;
    }

    private enqueueChunk(item: UploadQueueItem) {
        this.queue.push(item);
        // Small file first priority scheduling: sort by file size ascending
        this.queue.sort((a, b) => a.fileSize - b.fileSize);
        this.pump();
    }

    private async pump() {
        if (this.isPumping || this.isPaused) return;
        this.isPumping = true;

        try {
            while (this.queue.length > 0 && !this.isPaused) {
                // Find available worker
                const availableWorkers = this.workers;
                if (availableWorkers.length === 0) break;

                // Pick an item from queue
                const item = this.queue.shift();
                if (!item) break;

                const session = this.sessions.get(item.uploadId);
                if (!session || session.cancelled || session.shouldCancel?.()) {
                    continue;
                }

                // Choose worker in round-robin / hashing based on chunkIndex
                const worker = this.workers[item.chunkIndex % this.workers.length];
                this.inFlightMap.set(item.queueItemId, item);
                item.inFlight = true;

                // Fire worker sendFrame in background (parallel execution across workers)
                this.sendChunkViaWorker(worker, item).catch(err => {
                    console.error(`Failed to send chunk ${item.chunkIndex} via worker ${worker.id}:`, err);
                    this.handleSendFailure(item);
                });
            }
        } finally {
            this.isPumping = false;
        }
    }

    private async sendChunkViaWorker(worker: UploadWorkerConnection, item: UploadQueueItem) {
        const frameBuffer = await this.buildChunkFrame(item);
        await worker.sendFrame(frameBuffer);
    }

    private async buildChunkFrame(item: UploadQueueItem): Promise<ArrayBuffer> {
        const chunkBytes = item.payload;
        const checksum = await crypto.subtle.digest('SHA-256', chunkBytes.buffer as ArrayBuffer);
        const checksumHex = Array.from(new Uint8Array(checksum)).map(byte => byte.toString(16).padStart(2, '0')).join('');

        const encoder = new TextEncoder();
        const uploadIdBytes = encoder.encode(item.uploadId);
        const fileIdBytes = encoder.encode(item.fileId);
        const encodingBytes = encoder.encode(item.encoding);

        const headerLen = 1 + 4 + 8 + 4 + 4 + 4 + 64;
        const frame = new Uint8Array(headerLen + uploadIdBytes.length + fileIdBytes.length + encodingBytes.length + chunkBytes.length);
        const view = new DataView(frame.buffer);
        frame[0] = 1;
        view.setUint32(1, item.chunkIndex, true);
        view.setBigUint64(5, BigInt(chunkBytes.length), true);
        view.setUint32(13, uploadIdBytes.length, true);
        view.setUint32(17, fileIdBytes.length, true);
        view.setUint32(21, encodingBytes.length, true);

        let offset = 25;
        const checksumBytes = encoder.encode(checksumHex);
        frame.set(checksumBytes, offset);
        offset += 64;
        frame.set(uploadIdBytes, offset);
        offset += uploadIdBytes.length;
        frame.set(fileIdBytes, offset);
        offset += fileIdBytes.length;
        frame.set(encodingBytes, offset);
        offset += encodingBytes.length;
        frame.set(chunkBytes, offset);

        return frame.buffer.slice(frame.byteOffset, frame.byteOffset + frame.byteLength) as ArrayBuffer;
    }

    private handleAck(ack: { uploadId: string; fileId?: string; chunkIndex: number; payloadBytes: number }) {
        const key = `${ack.uploadId}:${ack.chunkIndex}`;
        this.inFlightMap.delete(key);

        const session = this.sessions.get(ack.uploadId);
        if (session) {
            session.tracker.recordChunkAcknowledged(ack.payloadBytes);
            this.emitSessionProgress(session, 'CHUNK_ACK');
            this.checkSessionCompletion(session);
        }

        this.pump();
    }

    private handleDataReceived(payload: any) {
        const uploadId = payload.uploadId;
        const session = this.sessions.get(uploadId);
        if (session) {
            session.tracker.markDataReceived();
            this.emitSessionProgress(session, 'DATA_RECEIVED');
        }
    }

    private handleFileComplete(payload: any) {
        const uploadId = payload.uploadId;
        const session = this.sessions.get(uploadId);
        if (session) {
            session.completed = true;
            session.tracker.markServerCompleted();
            this.emitSessionProgress(session, 'FILE_COMPLETE');
            session.onServerCompleted?.();
        }
    }

    private handleError(error: Error, uploadId?: string, _fileId?: string) {
        if (uploadId) {
            const session = this.sessions.get(uploadId);
            session?.onError?.(error);
        }
    }

    private handleSocketLost(workerId: number) {
        console.warn(`WebSocket worker ${workerId} disconnected. Reconnecting worker.`);
        // Re-queue any in-flight items assigned to this worker or reconnect
        const worker = this.workers.find(w => w.id === workerId);
        if (worker) {
            worker.ensureOpen().catch(err => console.error(`Worker ${workerId} reconnection failed:`, err));
        }
    }

    private handleSendFailure(item: UploadQueueItem) {
        this.inFlightMap.delete(item.queueItemId);
        if (item.retries < 5) {
            item.retries++;
            item.inFlight = false;
            this.queue.unshift(item); // Retry promptly
            setTimeout(() => this.pump(), 200);
        } else {
            const session = this.sessions.get(item.uploadId);
            session?.onError?.(new Error(`Failed to upload chunk ${item.chunkIndex} after ${item.retries} retries`));
        }
    }

    private checkSessionCompletion(session: ActiveFileUploadSession) {
        if (session.cancelled || session.completed) return;
        const snapshot = session.tracker.getSnapshot();

        if (canSendFinalize(snapshot)) {
            if (!session.dataReceivedNotified) {
                session.dataReceivedNotified = true;
                session.onDataReceived?.();
            }

            // Send FINALIZE through any available worker
            const worker = this.workers[0] || this.workers[1] || this.workers[2];
            if (worker) {
                worker.sendControl({
                    type: 'FINALIZE',
                    uploadId: session.uploadId,
                    fileId: session.selectableFile.uniqueId,
                    fileName: session.selectableFile.file.name,
                    totalChunks: snapshot.totalChunksGenerated,
                    encoding: 'gzip-stream-v1',
                    fileSize: session.selectableFile.file.size,
                    collectionId: session.collectionId,
                    auth: session.auth,
                });
            }
        }
    }

    private emitSessionProgress(session: ActiveFileUploadSession, reason: string) {
        const progress = session.tracker.getProgress();
        session.updateProgress(progress);
        if (session.tracker.isDataReceived() && !session.dataReceivedNotified) {
            session.dataReceivedNotified = true;
            session.onDataReceived?.();
        }
        const snapshot = session.tracker.getSnapshot();
        console.debug('[GLOBAL_UPLOAD_PROGRESS]', {
            file: session.selectableFile.file.name,
            uploadId: session.uploadId,
            event: reason,
            ackedChunks: snapshot.totalChunksAcknowledged,
            totalChunks: snapshot.totalChunksGenerated,
            ackedBytes: snapshot.acknowledgedCompressedBytes,
            totalCompressedBytes: snapshot.totalCompressedBytes,
            compressionFinished: snapshot.compressionFinished,
            dataReceived: snapshot.dataReceived,
            serverCompleted: snapshot.serverCompleted,
            progress,
        });
    }
}
