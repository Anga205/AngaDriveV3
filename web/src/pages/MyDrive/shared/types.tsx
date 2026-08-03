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

interface AuthDetails {
    token?: string;
    email?: string;
    password?: string;
}

/**
 * Upload encoding mode.
 *  - "gzip-stream-v1": chunks are gzip-compressed on the client (desktop).
 *  - "raw": chunks contain the original bytes (mobile/ARM).
 */
type UploadEncoding = "gzip-stream-v1" | "raw";

export type { SelectableFile, FileUploadProgressData, AuthDetails, UploadEncoding };