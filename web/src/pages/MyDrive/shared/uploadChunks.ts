import { generateClientToken } from "@/library/functions";
import type { SelectableFile, FileUploadProgressData, AuthDetails } from "./types";
import { apiUrl } from "@/assets/ApiUrl";

export const CHUNK_SIZE = 7 * 1024 * 1024; // 7MB chunk size
export const MAX_CONCURRENT_UPLOADS = 3;
export const MAX_CONCURRENT_CHUNKS_PER_FILE = 6;

export async function uploadFileInChunks(
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

export type { SelectableFile, FileUploadProgressData, AuthDetails };