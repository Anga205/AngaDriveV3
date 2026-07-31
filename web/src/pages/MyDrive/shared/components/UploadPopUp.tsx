import Dialog from "@corvu/dialog";
import { AppContext } from "@/Context";
import type { Component } from "solid-js"
import { createSignal, Show, For, createMemo, onCleanup, createEffect, useContext, onMount } from "solid-js"
import { UploadSVG } from "@/assets/SvgFiles"
import { toast } from 'solid-toast';
import { generateClientToken, generateUUID } from "@/library/functions";
import type { SelectableFile, FileUploadProgressData, AuthDetails } from "../types";
import FileUploadPreview from "./FileUploadPreview";
import { UploadProgressTracker, canSendFinalize } from "./uploadProgressState";

const CHUNK_SIZE = 7 * 1024 * 1024; // 7MB compressed chunk size
const MAX_CONCURRENT_UPLOADS = 3;
const PIPELINE_WINDOW_SIZE = 4; // Bounded window allowing up to 4 in-flight chunks

async function uploadFileInChunks(
    selectableFile: SelectableFile,
    uploadSystemId: string,
    authDetails: AuthDetails,
    updateProgress: (progress: number) => void,
    onDataReceived?: () => void,
    collectionId?: string,
    waitWhilePaused?: () => Promise<void>,
    isPaused?: () => boolean,
    shouldCancel?: () => boolean,
): Promise<void> {
    const file = selectableFile.file;
    const fileId = selectableFile.uniqueId;
    const encoding = 'gzip-stream-v1';
    const baseUrl = import.meta.env.DEV ? 'http://localhost:8080' : window.location.origin;
    const wsUrl = `${baseUrl.replace(/^http/, 'ws')}/upload/ws/${uploadSystemId}`;

    const tracker = new UploadProgressTracker(file.size);
    let chunkIndexCursor = 0;
    let serverCompleted = false;
    let dataReceivedNotified = false;
    let ws: WebSocket | null = null;
    let socketReady: Promise<void> | null = null;
    let socketReject: ((error: Error) => void) | null = null;
    const inFlightChunks = new Set<number>();
    const pendingChunkAcks = new Map<number, { resolve: () => void; reject: (error: Error) => void }>();
    const acknowledgedChunkIndexes = new Set<number>();
    const allAcksWaiters: Array<{ resolve: () => void; reject: (error: Error) => void }> = [];
    const windowAvailableWaiters: Array<() => void> = [];

    const emitProgress = (reason: string) => {
        const progress = tracker.getProgress();
        updateProgress(progress);
        if (tracker.isDataReceived() && !dataReceivedNotified) {
            dataReceivedNotified = true;
            onDataReceived?.();
        }
        const snapshot = tracker.getSnapshot();
        console.debug('[UPLOAD_PROGRESS]', {
            file: file.name,
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
    };

    const failPendingAcks = (error: Error) => {
        pendingChunkAcks.forEach((pending) => pending.reject(error));
        pendingChunkAcks.clear();
        inFlightChunks.clear();
        while (allAcksWaiters.length > 0) {
            allAcksWaiters.pop()?.reject(error);
        }
    };

    const notifyIfAllAcksSettled = () => {
        if (!canSendFinalize(tracker.getSnapshot())) return;
        while (allAcksWaiters.length > 0) {
            allAcksWaiters.pop()?.resolve();
        }
    };

    const notifyWindowSlot = () => {
        while (inFlightChunks.size < PIPELINE_WINDOW_SIZE && windowAvailableWaiters.length > 0) {
            const waiter = windowAvailableWaiters.shift();
            waiter?.();
        }
    };

    const waitForWindowSlot = async (): Promise<void> => {
        if (inFlightChunks.size < PIPELINE_WINDOW_SIZE) return;
        await new Promise<void>((resolve) => {
            windowAvailableWaiters.push(resolve);
        });
    };

    const waitForAllChunkAcks = async (): Promise<void> => {
        if (canSendFinalize(tracker.getSnapshot())) return;
        await new Promise<void>((resolve, reject) => {
            allAcksWaiters.push({ resolve, reject });
        });
    };

    const ensureSocket = (): Promise<WebSocket> => {
        if (ws && ws.readyState === WebSocket.OPEN) return Promise.resolve(ws);
        if (socketReady) return socketReady.then(() => ws as WebSocket);

        socketReady = new Promise((resolve, reject) => {
            socketReject = reject;
            ws = new WebSocket(wsUrl);
            ws.binaryType = 'arraybuffer';
            ws.onopen = () => {
                ws?.send(JSON.stringify({
                    type: 'INIT',
                    uploadId: uploadSystemId,
                    fileId,
                    fileName: file.name,
                    fileSize: file.size,
                    totalChunks: 0,
                    encoding,
                    auth: authDetails,
                }));
                resolve();
            };
            ws.onmessage = (event) => {
                const payload = typeof event.data === 'string' ? JSON.parse(event.data) : null;
                if (!payload) return;
                if (payload.type === 'CHUNK_ACK') {
                    const ackChunkIndex = Number(payload.chunkIndex ?? -1);
                    const ackBytes = Number(payload.payloadBytes || 0);
                    if (Number.isFinite(ackChunkIndex) && ackChunkIndex >= 0) {
                        inFlightChunks.delete(ackChunkIndex);
                        if (pendingChunkAcks.has(ackChunkIndex)) {
                            pendingChunkAcks.get(ackChunkIndex)?.resolve();
                            pendingChunkAcks.delete(ackChunkIndex);
                        }
                        notifyWindowSlot();
                    }
                    if (!acknowledgedChunkIndexes.has(ackChunkIndex)) {
                        acknowledgedChunkIndexes.add(ackChunkIndex);
                        tracker.recordChunkAcknowledged(Number.isFinite(ackBytes) && ackBytes > 0 ? ackBytes : 0);
                        emitProgress('CHUNK_ACK');
                    }
                    notifyIfAllAcksSettled();
                    return;
                }
                if (payload.type === 'DATA_RECEIVED') {
                    tracker.markDataReceived();
                    emitProgress('DATA_RECEIVED');
                    return;
                }
                if (payload.type === 'FILE_COMPLETE') {
                    serverCompleted = true;
                    tracker.markServerCompleted();
                    emitProgress('FILE_COMPLETE');
                    return;
                }
                if (payload.type === 'UPLOAD_ERROR') {
                    const error = new Error(payload.message || 'Upload error');
                    failPendingAcks(error);
                    socketReject?.(error);
                    throw error;
                }
            };
            ws.onerror = () => {
                const error = new Error('Upload socket error');
                failPendingAcks(error);
                socketReject?.(error);
            };
            ws.onclose = () => {
                if (!serverCompleted && !tracker.isDataReceived() && shouldCancel?.() !== true) {
                    const error = new Error('Upload socket closed before completion');
                    failPendingAcks(error);
                    socketReject?.(error);
                }
            };
        });

        return socketReady.then(() => ws as WebSocket);
    };

    const sendChunkFrame = async (chunkIndex: number, chunkBytes: Uint8Array): Promise<void> => {
        await waitForWindowSlot();
        const socket = await ensureSocket();
        // Fix: Cast buffer to ArrayBuffer to satisfy BufferSource type for crypto.subtle.digest
        const checksum = await crypto.subtle.digest('SHA-256', chunkBytes.buffer as ArrayBuffer);
        const checksumHex = Array.from(new Uint8Array(checksum)).map(byte => byte.toString(16).padStart(2, '0')).join('');

        const encoder = new TextEncoder();
        const uploadIdBytes = encoder.encode(uploadSystemId);
        const fileIdBytes = encoder.encode(fileId);
        const encodingBytes = encoder.encode(encoding);

        const headerLen = 1 + 4 + 8 + 4 + 4 + 4 + 64;
        const frame = new Uint8Array(headerLen + uploadIdBytes.length + fileIdBytes.length + encodingBytes.length + chunkBytes.length);
        const view = new DataView(frame.buffer);
        frame[0] = 1;
        view.setUint32(1, chunkIndex, true);
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

        if (socket.readyState !== WebSocket.OPEN) {
            throw new Error('Upload socket is not open');
        }
        inFlightChunks.add(chunkIndex);
        pendingChunkAcks.set(chunkIndex, {
            resolve: () => { },
            reject: () => { },
        });
        socket.send(frame.buffer.slice(frame.byteOffset, frame.byteOffset + frame.byteLength));
    };

    const compressionTask = async () => {
        const reader = file.stream().pipeThrough(new CompressionStream('gzip')).getReader();
        let buffer = new Uint8Array(0);
        while (true) {
            if (shouldCancel?.()) return;
            if (waitWhilePaused && isPaused && isPaused()) {
                await waitWhilePaused();
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
                tracker.recordChunkGenerated(slice.length);
                emitProgress('CHUNK_GENERATED');
                if (waitWhilePaused && isPaused && isPaused()) {
                    await waitWhilePaused();
                }
                await sendChunkFrame(chunkIndexCursor, slice);
                chunkIndexCursor += 1;
                if (shouldCancel?.()) return;
            }
        }

        if (buffer.length > 0) {
            tracker.recordChunkGenerated(buffer.length);
            emitProgress('FINAL_CHUNK_GENERATED');
            await sendChunkFrame(chunkIndexCursor, buffer);
            chunkIndexCursor += 1;
        }

        tracker.markCompressionFinished();
        emitProgress('COMPRESSION_FINISHED');
        await waitForAllChunkAcks();

        // All chunks ACKed -> DATA_RECEIVED on client side
        tracker.markDataReceived();
        emitProgress('ALL_CHUNKS_ACKED');

        if (shouldCancel?.()) return;
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: 'FINALIZE',
                uploadId: uploadSystemId,
                fileId,
                fileName: file.name,
                totalChunks: tracker.getSnapshot().totalChunksGenerated,
                encoding,
                fileSize: file.size,
                collectionId,
                auth: authDetails,
            }));
        }
    };

    try {
        await compressionTask();
        if (shouldCancel?.()) return;
        await new Promise<void>((resolve, reject) => {
            const interval = setInterval(() => {
                if (serverCompleted) {
                    clearInterval(interval);
                    resolve();
                    return;
                }
                if (shouldCancel?.()) {
                    clearInterval(interval);
                    reject(new Error('Upload cancelled'));
                }
            }, 50);
        });
    } catch (error: any) {
        throw new Error(error?.message || 'WebSocket upload failed');
    }
}

const UploadPopup: Component = () => {
    const ctx = useContext(AppContext)!;
    const [selectedFiles, setSelectedFiles] = createSignal<SelectableFile[]>([]);
    const [uploadProgressMap, setUploadProgressMap] = createSignal<Record<string, FileUploadProgressData>>({});
    const [isUploading, setIsUploading] = createSignal(false);
    const [isPaused, setIsPaused] = createSignal(false);
    const [isDragOver, setIsDragOver] = createSignal(false);
    const [open, setOpen] = createSignal(false);
    const activeControllers = new Set<AbortController>();
    // Track files removed during pause to cancel on resume
    const cancelledFiles = new Set<string>();

    const getPathKey = (f: File) => ((f as any).webkitRelativePath || (f as any).path || f.name);

    const addFiles = (files: FileList | File[]) => {
        const list = Array.from(files);
        if (list.length === 0) return;
        const existingKeys = new Set(selectedFiles().map(sf => getPathKey(sf.file)));
        const deduped = list
            .filter(f => !existingKeys.has(getPathKey(f)))
            .map(f => ({ uniqueId: generateUUID(), file: f }));
        if (deduped.length === 0) return;
        setSelectedFiles(prev => [...prev, ...deduped]);
        setUploadProgressMap(prevMap => {
            const updatedMap = { ...prevMap };
            deduped.forEach(sf => {
                if (!updatedMap[sf.uniqueId]) {
                    updatedMap[sf.uniqueId] = {
                        id: sf.uniqueId,
                        name: sf.file.name,
                        progress: 0,
                        status: 'pending',
                    };
                }
            });
            return updatedMap;
        });
        // Attempt to fill concurrency slots immediately when not paused
        if (!isPaused()) queueMicrotask(() => pumpQueue());
    };

    const handleFileChange = (event: Event) => {
        const input = event.target as HTMLInputElement;
        if (input.files) {
            addFiles(input.files);
        }
        if (input) input.value = ''; // Reset input
    };

    const addDroppedFiles = (files: FileList | File[]) => {
        addFiles(files);
        if (open() && !isPaused()) queueMicrotask(() => pumpQueue());
    };

    const handleFileDelete = (uniqueIdToDelete: string) => {
        cancelledFiles.add(uniqueIdToDelete);
        setSelectedFiles((prev) => prev.filter(sf => sf.uniqueId !== uniqueIdToDelete));
        setUploadProgressMap(prev => {
            const updated = { ...prev };
            delete updated[uniqueIdToDelete];
            return updated;
        });
    };

    const hasActiveUploads = createMemo(() => {
        const map = uploadProgressMap();
        return selectedFiles().some(sf => {
            const status = map[sf.uniqueId]?.status;
            return status === 'uploading' || status === 'pending';
        });
    });

    const allUploadsComplete = createMemo(() => {
        const files = selectedFiles();
        if (files.length === 0) return false;

        const map = uploadProgressMap();
        return files.every(sf => {
            const status = map[sf.uniqueId]?.status;
            return status === 'completed' || status === 'processing' || status === 'error';
        });
    });

    const canClosePopup = createMemo(() => !hasActiveUploads() || isPaused());
    const shouldPreserveUploadStateOnClose = createMemo(() => isPaused() && selectedFiles().length > 0 && !allUploadsComplete());

    const handleDialogStateChange = (isOpen: boolean) => {
        if (!isOpen) {
            if (!canClosePopup()) {
                toast.error('Uploads are still in progress. Pause them before closing the upload popup.');
                setOpen(true);
                return;
            }

            if (shouldPreserveUploadStateOnClose()) {
                return;
            }

            // Reset when dialog closes
            setSelectedFiles([]);
            setUploadProgressMap({});
            setIsUploading(false);
            setIsPaused(false);
            setIsDragOver(false);
            // Abort any lingering requests
            activeControllers.forEach(c => c.abort());
            activeControllers.clear();
            cancelledFiles.clear();
        }
    };

    onMount(() => {
        const handler = (e: Event) => {
            const ce = e as CustomEvent<{ files?: File[] | FileList }>;
            const files = ce.detail?.files;
            if (files && (files as any).length !== undefined) {
                addDroppedFiles(files as File[] | FileList);
            }
            setOpen(true);
        };
        document.addEventListener("open-drive-upload", handler as EventListener);
        // If navigation stored pending files, consume them (only if not already open)
        queueMicrotask(() => {
            if (!open()) {
                try {
                    const pending = ctx.pendingDriveUploadFiles?.() || null;
                    if (pending && pending.length) {
                        addDroppedFiles(pending);
                        ctx.setPendingDriveUploadFiles?.(null);
                        setOpen(true);
                    }
                } catch (err) {
                    console.error('Error consuming pending drive upload files from context:', err);
                }
            }
        });
        onCleanup(() => document.removeEventListener("open-drive-upload", handler as EventListener));
    });

    // Concurrency-aware scheduler that always fills up to MAX_CONCURRENT_UPLOADS slots
    let activeUploads = 0;

    const waitWhilePaused = async () => {
        while (isPaused()) {
            await new Promise(r => setTimeout(r, 200));
        }
    };

    const buildAuthDetails = (): AuthDetails | null => {
        const storedEmail = localStorage.getItem("email");
        const storedPassword = localStorage.getItem("password");
        if (storedEmail && storedPassword) {
            return { email: storedEmail, password: storedPassword };
        }
        let token = localStorage.getItem("token");
        if (!token) {
            token = generateClientToken();
            localStorage.setItem("token", token);
        }
        if (!token) return null;
        return { token };
    };

    const startSingleUpload = async (selectableFile: SelectableFile) => {
        // Skip if cancelled
        if (cancelledFiles.has(selectableFile.uniqueId)) return;

        const authDetails = buildAuthDetails();
        if (!authDetails) {
            // Mark as error for missing auth
            setUploadProgressMap(prev => ({
                ...prev,
                [selectableFile.uniqueId]: {
                    ...prev[selectableFile.uniqueId],
                    status: 'error',
                    errorMessage: 'Authentication configuration error',
                }
            }));
            return;
        }

        // Transition to uploading
        setUploadProgressMap(prev => {
            if (cancelledFiles.has(selectableFile.uniqueId)) return prev;
            return ({
                ...prev,
                [selectableFile.uniqueId]: {
                    ...prev[selectableFile.uniqueId],
                    status: 'uploading',
                    // Keep current progress if retrying; otherwise ensure 0
                    progress: prev[selectableFile.uniqueId]?.progress ?? 0,
                }
            });
        });

        const backendUploadId = generateUUID();
        activeUploads++;
        try {
            await uploadFileInChunks(
                selectableFile,
                backendUploadId,
                authDetails,
                (progress) => {
                    setUploadProgressMap(prev => {
                        if (cancelledFiles.has(selectableFile.uniqueId)) return prev;
                        return ({
                            ...prev,
                            [selectableFile.uniqueId]: {
                                ...prev[selectableFile.uniqueId],
                                progress,
                            }
                        });
                    });
                },
                () => {
                    // onDataReceived callback: server has all data, 100% upload complete
                    setUploadProgressMap(prev => {
                        if (cancelledFiles.has(selectableFile.uniqueId)) return prev;
                        return ({
                            ...prev,
                            [selectableFile.uniqueId]: {
                                ...prev[selectableFile.uniqueId],
                                status: prev[selectableFile.uniqueId]?.status === 'completed' ? 'completed' : 'processing',
                                progress: 100,
                            }
                        });
                    });
                },
                undefined,
                waitWhilePaused,
                () => isPaused(),
                () => cancelledFiles.has(selectableFile.uniqueId)
            );
            setUploadProgressMap(prev => {
                if (cancelledFiles.has(selectableFile.uniqueId)) return prev;
                return ({
                    ...prev,
                    [selectableFile.uniqueId]: {
                        ...prev[selectableFile.uniqueId],
                        status: 'completed',
                        progress: 100,
                    }
                });
            });
        } catch (error: any) {
            console.error(`Error uploading file ${selectableFile.file.name}:`, error);
            setUploadProgressMap(prev => {
                if (cancelledFiles.has(selectableFile.uniqueId)) return prev;
                return ({
                    ...prev,
                    [selectableFile.uniqueId]: {
                        ...prev[selectableFile.uniqueId],
                        status: 'error',
                        errorMessage: error?.name === 'AbortError' ? 'Paused' : (error?.message || 'Upload failed'),
                    }
                });
            });
        } finally {
            activeUploads--;
            // If not paused, try to fill freed slot
            queueMicrotask(() => pumpQueue());
        }
    };

    const pumpQueue = () => {
        if (!open()) return; // Only process when dialog open
        if (isPaused()) return; // Don't start new uploads while paused

        // Ensure map has pending entries for any new files (in case of retries)
        setUploadProgressMap(prev => {
            const updated = { ...prev };
            selectedFiles().forEach(sf => {
                if (!updated[sf.uniqueId]) {
                    updated[sf.uniqueId] = {
                        id: sf.uniqueId,
                        name: sf.file.name,
                        progress: 0,
                        status: 'pending',
                    };
                }
            });
            return updated;
        });

        const currentMap = uploadProgressMap();
        const availableSlots = Math.max(0, MAX_CONCURRENT_UPLOADS - activeUploads);
        if (availableSlots === 0) {
            setIsUploading(true);
            return;
        }

        const candidates = [...selectedFiles()]
            .filter(sf => {
                const st = currentMap[sf.uniqueId]?.status;
                return (st === 'pending' || st === 'error') && !cancelledFiles.has(sf.uniqueId);
            })
            .sort((a, b) => a.file.size - b.file.size)
            .slice(0, availableSlots);

        if (candidates.length === 0) {
            // Nothing to start. If none running either, mark idle.
            if (activeUploads === 0) setIsUploading(false);
            return;
        }

        setIsUploading(true);
        candidates.forEach(sf => {
            // Reset to pending (clears error state for retry) before starting
            setUploadProgressMap(prev => ({
                ...prev,
                [sf.uniqueId]: {
                    ...prev[sf.uniqueId],
                    status: 'pending',
                    progress: prev[sf.uniqueId]?.status === 'error' ? 0 : (prev[sf.uniqueId]?.progress ?? 0),
                    errorMessage: undefined,
                }
            }));
            // Kick off the upload
            startSingleUpload(sf);
        });
    };

    const filesPendingOrError = createMemo(() => {
        return selectedFiles().filter(sf => {
            const status = uploadProgressMap()[sf.uniqueId]?.status;
            return status === 'pending' || status === 'error';
        }).length;
    });

    const anyUploading = createMemo(() => {
        return selectedFiles().some(sf => uploadProgressMap()[sf.uniqueId]?.status === 'uploading');
    });

    // Auto-start and keep pumping while there are pending items and not paused
    createEffect(() => {
        if (open() && selectedFiles().length > 0 && filesPendingOrError() > 0 && !isPaused()) {
            queueMicrotask(() => pumpQueue());
        }
    });


    return (
        <Dialog open={open()} onOpenChange={(o) => { setOpen(o); handleDialogStateChange(o); }}>
            <Dialog.Trigger class="h-full min-w-fit flex items-center justify-center gap-2 cursor-pointer hover:text-neutral-300 text-white bg-blue-600 hover:bg-blue-800 p-[1vh] rounded-[0.6vh] font-bold">
                <UploadSVG />
                <span>&nbsp;Upload</span>
            </Dialog.Trigger>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 z-50 bg-black/50 data-open:animate-in data-open:fade-in-0% data-closed:animate-out data-closed:fade-out-0%" />
                <Dialog.Content class="bg-[#0f0f0f] text-white fixed left-1/2 top-1/2 z-50 min-w-[clamp(320px,80vw,600px)] w-auto max-w-[90vw] -translate-x-1/2 -translate-y-1/2 rounded-lg border-2 border-neutral-900 bg-corvu-100 px-6 py-5 data-open:animate-in data-open:fade-in-0% data-open:zoom-in-95% data-open:slide-in-from-top-10% data-closed:animate-out data-closed:fade-out-0% data-closed:zoom-out-95% data-closed:slide-out-to-top-10%">
                    <Dialog.Label class="text-lg font-bold">
                        Upload Files
                    </Dialog.Label>
                    <label
                        for="file-upload"
                        class={`rounded-md min-h-[15vh] flex justify-center items-center cursor-pointer my-[1vh] ${selectedFiles().length === 0 ? `border-2 ${isDragOver() ? 'border-blue-400' : 'border-dotted border-blue-800'}` : ''}`}
                        onDragOver={(e) => { e.preventDefault(); setIsDragOver(true); }}
                        onDragEnter={(e) => { e.preventDefault(); setIsDragOver(true); }}
                        onDragLeave={() => setIsDragOver(false)}
                        onDrop={(e) => {
                            e.preventDefault();
                            setIsDragOver(false);
                            if (e.dataTransfer?.files && e.dataTransfer.files.length > 0) {
                                addDroppedFiles(e.dataTransfer.files);
                            }
                        }}
                    >
                        {selectedFiles().length === 0 ? (
                            <p class="text-center p-4">Drag and drop files here or click to select files</p>
                        ) : (
                            <div class="w-full space-y-2 max-h-[50vh] overflow-y-auto custom-scrollbar p-1">
                                <div class="w-full space-y-2">
                                    <For each={selectedFiles()}>
                                        {(sf) => (
                                            <FileUploadPreview
                                                selectableFile={sf}
                                                uploadInfo={() => uploadProgressMap()[sf.uniqueId]}
                                                onDelete={handleFileDelete}
                                                canDelete={() => isPaused()}
                                            />
                                        )}
                                    </For>
                                </div>
                            </div>
                        )}
                        <input id="file-upload" type="file" multiple class="hidden" onChange={handleFileChange} />
                    </label>
                    <Show when={selectedFiles().length > 0 && (filesPendingOrError() > 0 || anyUploading())}>
                        <button
                            class={`mt-4 w-full ${isPaused() ? 'bg-blue-600 hover:bg-blue-800' : 'bg-yellow-600 hover:bg-yellow-800'} disabled:bg-neutral-600 text-white font-bold py-2 px-4 rounded`}
                            onClick={() => {
                                const next = !isPaused();
                                setIsPaused(next);
                                if (next) {
                                    // Pausing: abort all in-flight uploads to pause immediately
                                    activeControllers.forEach(c => c.abort());
                                } else {
                                    // Resuming: fill available slots immediately
                                    queueMicrotask(() => pumpQueue());
                                }
                            }}
                            disabled={false}
                        >
                            {isPaused() ? 'Resume' : 'Pause'}
                        </button>
                    </Show>
                    <Show when={selectedFiles().length > 0 && filesPendingOrError() === 0 && !isUploading() && allUploadsComplete()}>
                        <p class="mt-4 text-center text-green-500">All selected files uploaded!</p>
                    </Show>
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    );
};

export { UploadPopup, uploadFileInChunks };