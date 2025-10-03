import type { Accessor, Component } from "solid-js"
import { createSignal, Show, For, createMemo, onCleanup, createEffect, useContext, onMount } from "solid-js"
import { DesktopTemplate } from "../components/Template"
import { InfoSVG, UploadSVG, ErrorSVG, BinSVG, FileSVG } from "../assets/SvgFiles"
import Navbar from "../components/Navbar";
import { useWebSocket } from "../Websockets";
import { FileData } from "../library/types";
import Dialog from "@corvu/dialog";
import FileCard from "../components/FileCard";
import { Toaster, toast } from 'solid-toast';
import { AppContext } from "../Context";
import { formatFileSize, truncateFileName, generateClientToken, generateUUID } from "../library/functions";

const FilesError: Component = () => {
    const baseClass = "flex items-center p-[1vh] rounded-[1vh] w-full";
    const textClass = "text-sm";
    const containerClass = "md:max-w-1/3 flex flex-col items-center space-y-[2vh]";
    const {status} = useWebSocket();

    return (
        <div class="w-full h-full flex justify-center items-center px-10 md:px-0">
            <div class={containerClass}>
                {((status() === "connecting") || (status() === "reconnecting") ) && (
                    <div class={`${baseClass} border-l-[0.2vw] bg-yellow-600/30 border-yellow-400`}>
                        <div class="pr-[0.75vw] text-yellow-600 w-16 md:w-[3vw]">
                            <InfoSVG />
                        </div>
                        <div>
                            <p class={`${textClass} text-yellow-400`}>Connecting to backend, please wait...</p>
                        </div>
                    </div>
                )}
                {status() === "connected" && (
                    <div class={`${baseClass} border-l-[0.2vw] bg-blue-600/30 border-blue-400`}>
                        <div class="pr-[0.75vw] text-blue-600 w-16 md:w-[3vw]">
                            <InfoSVG />
                        </div>
                        <div>
                            <p class={`${textClass} text-blue-400`}>Any files you upload will show up here, click on the &apos;Upload&apos; button to start uploading files or Drag & Drop files anywhere on this website</p>
                        </div>
                    </div>
                )}
                {(status() === "error" || status() === "disconnected") && (
                    <div class={`${baseClass} border-1 bg-red-600/30 border-red-400`}>
                        <div class="pr-[0.75vw] text-red-600 w-16 md:w-[3vw]">
                            <ErrorSVG />
                        </div>
                        <div>
                            <p class={`${textClass} text-red-400`}>Failed to connect to server</p>
                        </div>
                    </div>
                )}
            </div>
        </div>
    )
}



const DriveBody: Component<{ Files: Accessor<Array<FileData>> }> = (props) => {
    return (
        <div 
            class="w-full flex justify-center flex-wrap h-full space-x-8 space-y-8 overflow-y-scroll pt-10 custom-scrollbar"
        >
            <For each={props.Files()}>
            {(file) => <FileCard File={file} />}
            </For>
        </div>
    );
};

interface SelectableFile {
    uniqueId: string;
    file: File;
}

interface FileUploadProgressData {
    id: string; // Corresponds to SelectableFile.uniqueId
    name: string;
    progress: number; // 0-100
    status: 'pending' | 'uploading' | 'completed' | 'error';
    errorMessage?: string;
}

const FileUploadPreview: Component<{
    selectableFile: SelectableFile;
    uploadInfo?: Accessor<FileUploadProgressData | undefined>;
    onDelete: (uniqueId: string) => void;
    canDelete?: Accessor<boolean>;
}> = (props) => {

    const preview_size_limit = 100 * 1024 * 1024; // 100 MB
    const ext = props.selectableFile.file.name.split('.').pop()?.toLowerCase();
    const file = props.selectableFile.file;

    const info = createMemo(() => props.uploadInfo ? props.uploadInfo() : undefined);
    const [objectUrl, setObjectUrl] = createSignal<string | undefined>();

    createMemo(() => {
        if (ext && (["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"].includes(ext) || ["mp4", "mkv", "mov", "wmv", "flv", "webm"].includes(ext)) && file.size < preview_size_limit) {
            const url = URL.createObjectURL(file);
            setObjectUrl(url);
            onCleanup(() => URL.revokeObjectURL(url));
        } else {
            setObjectUrl(undefined);
        }
    });

    return (
        <div class="flex items-center justify-between p-3 bg-gray-900 rounded-lg shadow-lg hover:shadow-xl transition-shadow">
            <div class="flex items-center space-x-4">
                <div class="w-12 h-12 flex items-center justify-center bg-gray-800 rounded-md overflow-hidden">
                    {(() => {
                        const url = objectUrl();
                        if (ext && ["pdf"].includes(ext)) {
                            return <p class="text-white text-xs">PDF</p>;
                        }
                        if (url && ext && ["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"].includes(ext)) {
                            return <img src={url} alt="Preview" class="w-full h-full object-cover" />;
                        }
                        if (url && ext && ["mp4", "mkv", "mov", "wmv", "flv", "webm"].includes(ext)) {
                            return <video src={url} class="w-full h-full object-cover" muted />;
                        }
                        return <FileSVG />;
                    })()}
                </div>
                <div class="flex flex-col">
                    <p class="text-white text-sm font-medium">{truncateFileName(file.name)}</p>
                    <p class="text-gray-400 text-xs">{formatFileSize(file.size)}</p>
                </div>
            </div>
            <div class="flex-grow ml-4 min-w-[100px]">
                <Show when={info() && (info()!.status === 'uploading' || info()!.status === 'pending')}>
                    <div class="w-full bg-gray-700 rounded-full h-2.5">
                        <div
                            class="bg-blue-600 h-2.5 rounded-full transition-all duration-100 ease-linear"
                            style={{ width: `${info() ? info()!.progress : 0}%` }}
                        ></div>
                    </div>
                    <p class="text-xs text-gray-400 mt-1 text-right">{info()!.status === 'pending' ? 'Pending...' : `${info()!.progress}%`}</p>
                </Show>
                <Show when={info() && info()!.status === 'completed'}>
                    <p class="text-sm font-semibold text-green-500 text-right">Uploaded!</p>
                </Show>
                <Show when={info() && info()!.status === 'error'}>
                    <p class="text-sm font-semibold text-red-500 text-right" title={info()!.errorMessage}>Error</p>
                </Show>
            </div>
            <button
                class="flex items-center justify-center p-2 ml-2 bg-red-700/30 hover:bg-red-700/20 rounded-lg text-red-500 hover:text-red-700 transition-colors"
                onClick={() => props.onDelete(props.selectableFile.uniqueId)}
                disabled={props.canDelete ? !props.canDelete() : (!!info() && info()!.status === 'uploading')}
            >
                <BinSVG />
            </button>
        </div>
    )
}

const CHUNK_SIZE = 7 * 1024 * 1024; // 7MB chunk size
const MAX_CONCURRENT_UPLOADS = 3;
const MAX_CONCURRENT_CHUNKS_PER_FILE = 6;

interface AuthDetails {
    token?: string;
    email?: string;
    password?: string;
}

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
    
    const backendUrl = import.meta.env.DEV ? 'http://localhost:8080' : '';
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
            const response = await fetch(`${backendUrl}/upload/${uploadSystemId}`, {
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

    const successResponse = await fetch(`${backendUrl}/upload/success/${uploadSystemId}`, {
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
            const retryResponse = await fetch(`${backendUrl}/upload/success/${uploadSystemId}`, {
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
    
    const handleDialogStateChange = (isOpen: boolean) => {
        if (!isOpen) { // Reset when dialog closes, or on open if preferred
            setSelectedFiles([]);
            setUploadProgressMap({});
            setIsUploading(false);
            setIsPaused(false);
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
                setOpen(true);
                addDroppedFiles(files as File[] | FileList);
            }
        };
        document.addEventListener("open-drive-upload", handler as EventListener);
        // If navigation stored pending files, consume them (only if not already open)
        queueMicrotask(() => {
            if (!open()) {
                const pending = (window as any).__pendingDriveUploadFiles as File[] | undefined;
                if (pending && pending.length) {
                    setOpen(true);
                    addDroppedFiles(pending);
                    (window as any).__pendingDriveUploadFiles = undefined;
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
            <Dialog.Trigger class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-blue-600 hover:bg-blue-800 p-[1vh] rounded-[1vh] font-bold md:translate-y-[4vh]">
                <UploadSVG />
                <span>&nbsp;Upload</span>
            </Dialog.Trigger>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 z-50 bg-black/50 data-open:animate-in data-open:fade-in-0% data-closed:animate-out data-closed:fade-out-0%" />
                <Dialog.Content class="bg-[#0f0f0f] text-white fixed left-1/2 top-1/2 z-50 min-w-[clamp(320px,80vw,600px)] w-auto max-w-[90vw] -translate-x-1/2 -translate-y-1/2 rounded-lg border-2 border-gray-900 bg-corvu-100 px-6 py-5 data-open:animate-in data-open:fade-in-0% data-open:zoom-in-95% data-open:slide-in-from-top-10% data-closed:animate-out data-closed:fade-out-0% data-closed:zoom-out-95% data-closed:slide-out-to-top-10%">
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
                            <div class="flex flex-col w-full space-y-2 max-h-[50vh] overflow-y-auto custom-scrollbar p-1">
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
                        )}
                        <input id="file-upload" type="file" multiple class="hidden" onChange={handleFileChange} />
                    </label>
                    <Show when={selectedFiles().length > 0 && (filesPendingOrError() > 0 || anyUploading())}>
                        <button
                            class={`mt-4 w-full ${isPaused() ? 'bg-blue-600 hover:bg-blue-800' : 'bg-yellow-600 hover:bg-yellow-800'} disabled:bg-gray-600 text-white font-bold py-2 px-4 rounded`}
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
                     <Show when={selectedFiles().length > 0 && filesPendingOrError() === 0 && !isUploading()}>
                        <p class="mt-4 text-center text-green-500">All selected files uploaded!</p>
                    </Show>
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    );
};

const DesktopDrive: Component<{Files: Accessor<Array<FileData>>}> = (props) => {
    return (
        <DesktopTemplate CurrentPage="Files">
            <div class="flex flex-col w-full h-full px-[2vh] p-[1vh] space-y-10">
                <div class="w-full flex justify-between items-center">
                    <p class="text-white font-black text-[4vh]">My Files</p>
                    <UploadPopup />
                </div>
                {props.Files().length === 0 ? <FilesError /> : <DriveBody Files={props.Files} />}
            </div>
        </DesktopTemplate>
    )
}

const MobileDrive: Component<{Files: Accessor<Array<FileData>>}> = (props) => {
    return (
        <div class="flex flex-col w-full max-h-screen h-screen bg-black">
            <Navbar CurrentPage="Files" Type="mobile"/>
            <div class="h-[6vh]"/>
            <p class="text-white font-black text-[4vh] px-3">My&nbsp;Files</p>
            <div class="flex justify-end px-3">
                <UploadPopup/>
            </div>
            <div class="w-full px-4 mt-4 max-h-full h-full flex flex-wrap items-center space-y-4 space-x-4 justify-center overflow-y-auto">
                <For each={props.Files()} fallback={<FilesError />}>
                    {(file) => (
                        <FileCard File={file} />
                    )}
                </For>
                <div class="w-full h-[2vh]"/>
            </div>
        </div>
    )
}

const MyDrive: Component = () => {
    const [isMobile, setIsMobile] = createSignal(window.innerWidth <= 768);

    const handleResize = () => {
        setIsMobile(window.innerWidth <= 768);
    };
    window.addEventListener('resize', handleResize);
    onCleanup(() => window.removeEventListener('resize', handleResize));

    const ctx = useContext(AppContext)!;

    const { socket: getSocket } = useWebSocket();

    const messageHandler = (event: MessageEvent) => {
        const data = JSON.parse(event.data);
        if (data.type === "convert_video_response") {
            if (data.data.error) {
                toast.error(`${data.data.error}`);
            } else if (data.data.file) {
                toast.success(`Video converted successfully: ${data.data.file.original_file_name}`);
            }
        } else if (data.type === "delete_file_response") {
            if (data.data.error) {
                toast.error(`Error deleting file: ${data.data.error}`);
            } else {
                toast.success(`File deleted successfully: ${data.data.success}`);
            }
        }
    }


    createEffect(() => {
        const socket = getSocket();
        if (!socket) return;
        socket.addEventListener("message", messageHandler);
        onCleanup(() => {
            socket.removeEventListener("message", messageHandler);
        });
    })


    return (
        <>
            <title>My Files | DriveV3</title>
            {isMobile() ? <MobileDrive Files={ctx.files}/> : <DesktopDrive Files={ctx.files}/>}
            <Toaster
            position="bottom-right"
            gutter={8}
            containerClassName=""
            containerStyle={{}}
            toastOptions={{
                className: '',
                duration: 2000,
                style: {
                background: '#363636',
                color: '#fff',
                },
            }}
            />
        </>
    )
}

export { MyDrive, uploadFileInChunks, FileUploadPreview }
export type { SelectableFile, FileUploadProgressData }