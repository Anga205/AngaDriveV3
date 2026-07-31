interface SelectableFile {
    uniqueId: string;
    file: File;
}

interface FileUploadProgressData {
    id: string; // Corresponds to SelectableFile.uniqueId
    name: string;
    progress: number; // 0-100
    status: 'pending' | 'uploading' | 'processing' | 'completed' | 'error';
    errorMessage?: string;
}

interface AuthDetails {
    token?: string;
    email?: string;
    password?: string;
}

export type { SelectableFile, FileUploadProgressData, AuthDetails };