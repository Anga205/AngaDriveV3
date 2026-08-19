import Dialog from "@corvu/dialog";
import { AppContext } from "@/Context";
import type { Component } from "solid-js"
import { createSignal, Show, For, createMemo, onCleanup, createEffect, useContext, onMount } from "solid-js"
import { UploadSVG} from "@/assets/SvgFiles"
import { toast } from 'solid-toast';
import { generateClientToken, generateUUID } from "@/library/functions";
import type { SelectableFile, FileUploadProgressData, AuthDetails } from "../types";
import FileUploadPreview from "./FileUploadPreview";
import { apiUrl } from "@/assets/ApiUrl";

const CHUNK_SIZE = 7 * 1024 * 1024; // 7MB chunk size
const MAX_CONCURRENT_UPLOADS = 3;
const MAX_CONCURRENT_CHUNKS_PER_FILE = 6;

async function uploadFileInChunks(
    selectableFile: SelectableFile,
    uploadSystemId: string,
    authDetails: AuthDetails,
    updateProgress: (progress: number) => void,
    collectionId?: string,
    waitWhilePaused?: () => Promise<void>,
    isPaused?: () => boolean,
    manageController?: (c: AbortController, action: 'add' | 'remove') => void,
    shouldCancel?: () => boolean,
): Promise<void> {
    const file = selectableFile.file;
    const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
    let uploadedChunks = 0;
    const chunkQueue = Array.from({ length: totalChunks }, (_, i) => i);

    const uploadChunk = async (chunkIndex: number): Promise<void> => {
        if (shouldCancel && shouldCancel()) return;
        const start = chunkIndex * CHUNK_SIZE;
        const end = Math.min(start + CHUNK_SIZE, file.size);
        const chunkBlob = file.slice(start, end);

        // Compress the chunk using the Compression Streams API
        const stream = new Blob([chunkBlob]).stream().pipeThrough(new CompressionStream('gzip'));
        const compressedBlob = await new Response(stream).blob();

        const formData = new FormData();
        formData.append('chunk', compressedBlob, `${file.name}.gz`);
        formData.append('chunkIndex', String(chunkIndex));

        const controller = new AbortController();
        try {
            if (manageController) manageController(controller, 'add');
            const response = await fetch(apiUrl(`/upload/${uploadSystemId}`), {
                method: 'POST',
                body: formData,
                signal: controller.signal,
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`Chunk ${chunkIndex} upload failed (${response.status}): ${errorText}`);
            }
            // This needs to be atomic for concurrent updates
            const newUploadedCount = uploadedChunks + 1;
            uploadedChunks = newUploadedCount;
            updateProgress(Math.round((newUploadedCount / totalChunks) * 100));
        } finally {
            if (manageController) manageController(controller, 'remove');
        }
    };

    const worker = async (): Promise<void> => {
        while (chunkQueue.length > 0) {
            if (shouldCancel && shouldCancel()) return;
            if (waitWhilePaused && isPaused && isPaused()) {
                await waitWhilePaused();
            }
            const chunkIndex = chunkQueue.shift();
            if (chunkIndex === undefined) {
                break;
            }
            try {
                if (shouldCancel && shouldCancel()) return;
                await uploadChunk(chunkIndex);
            } catch (e: any) {
                // If paused and a request was aborted, re-enqueue this chunk to retry after resume
                if ((isPaused && isPaused()) && (e?.name === 'AbortError' || /aborted/i.test(String(e?.message || '')))) {
                    chunkQueue.unshift(chunkIndex);
                    if (waitWhilePaused) await waitWhilePaused();
                    continue;
                }
                throw e;
            }
        }
    };

    const uploadPromises: Promise<void>[] = [];
    for (let i = 0; i < Math.min(MAX_CONCURRENT_CHUNKS_PER_FILE, totalChunks); i++) {
        uploadPromises.push(worker());
    }
    await Promise.all(uploadPromises);

    // If cancelled, don't finalize; exit silently
    if (shouldCancel && shouldCancel()) {
        return;
    }

    if (uploadedChunks !== totalChunks) {
        throw new Error("Not all chunks were uploaded successfully.");
    }

    let finalizeFormData = new FormData();
    finalizeFormData.append('totalChunks', String(totalChunks));
    finalizeFormData.append('originalFileName', file.name);
    if (collectionId) {
        finalizeFormData.append('collectionId', collectionId);
    }

    if (authDetails.token) {
        finalizeFormData.append('token', authDetails.token);
    } else if (authDetails.email && authDetails.password) {
        finalizeFormData.append('email', authDetails.email);
        finalizeFormData.append('password', authDetails.password);
    } else {
        throw new Error("No authentication details provided for finalization.");
    }

    const successResponse = await fetch(apiUrl(`/upload/success/${uploadSystemId}`), {
        method: 'POST',
        body: finalizeFormData,
    });

    if (!successResponse.ok) {
        let responseText = await successResponse.text();
        let errorData;
        try {
            errorData = JSON.parse(responseText);
        } catch {
            errorData = { message: `Finalization failed with status ${successResponse.status}: ${responseText}` };
        }
        if ((successResponse.status === 401) && (responseText === "Invalid email or password")) {
            localStorage.removeItem("email");
            localStorage.removeItem("password");
            localStorage.removeItem("display_name");
            if (!localStorage.getItem("token")) {
                localStorage.setItem("token", generateClientToken());
            }
            finalizeFormData = new FormData();
            finalizeFormData.append('totalChunks', String(totalChunks));
            finalizeFormData.append('originalFileName', file.name);
            finalizeFormData.append('token', localStorage.getItem("token") || "");
            if (collectionId) {
                finalizeFormData.append('collectionId', collectionId);
            }
            const retryResponse = await fetch(apiUrl(`/upload/success/${uploadSystemId}`), {
                method: 'POST',
                body: finalizeFormData,
            });
            if (!retryResponse.ok) {
                let retryResponseText = await retryResponse.text();
                let retryErrorData;
                try {
                    retryErrorData = JSON.parse(retryResponseText);
                } catch {
                    retryErrorData = { message: `Retry finalization failed with status ${retryResponse.status}: ${retryResponseText}` };
                }
                throw new Error(`Retry finalization failed: ${retryErrorData.message || retryResponse.statusText}`);
            }
            return; // Successfully retried finalization
        }
        throw new Error(`Finalization failed: ${errorData.message || successResponse.statusText}`);
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
    // Track active controllers to cancel on pause
    const activeControllers = new Set<AbortController>();
    const manageController = (c: AbortController, action: 'add' | 'remove') => {
        if (action === 'add') activeControllers.add(c);
        else activeControllers.delete(c);
    };
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
            return status === 'completed' || status === 'error';
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
                undefined,
                waitWhilePaused,
                () => isPaused(),
                manageController,
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

        const candidates = selectedFiles().filter(sf => {
            const st = currentMap[sf.uniqueId]?.status;
            return (st === 'pending' || st === 'error') && !cancelledFiles.has(sf.uniqueId);
        }).slice(0, availableSlots);

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
        <Dialog open={open()} onOpenChange={(o)=>{ setOpen(o); handleDialogStateChange(o); }}>
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