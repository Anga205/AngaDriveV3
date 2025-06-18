import type { Accessor, Component } from "solid-js"
import { createSignal, Show, For, createMemo, onCleanup, createEffect, useContext } from "solid-js"
import { DesktopTemplate } from "../components/Template"
import { InfoSVG, UploadSVG, ErrorSVG, BinSVG, FileSVG } from "../assets/SvgFiles"
import Navbar from "../components/Navbar";
import { useWebSocket } from "../Websockets";
import { FileData } from "../library/types";
import Dialog from "@corvu/dialog";
import FileCard from "../components/FileCard";
import { Toaster } from 'solid-toast';
import toast from "solid-toast";
import { AppContext } from "../Context";
import { formatFileSize, getFileMD5, truncateFileName } from "../library/functions";

const FilesError: Component = () => {
    const baseClass = "flex items-center p-[1vh] rounded-[1vh] w-full";
    const textClass = "text-[0.75vw]";
    const containerClass = "md:max-w-1/3 flex flex-col items-center space-y-[2vh]";
    const {status} = useWebSocket();

    return (
        <div class="w-full h-full flex justify-center items-center px-10 md:px-0">
            <div class={containerClass}>
                {((status() === "connecting") || (status() === "reconnecting") ) && (
                    <div class={`${baseClass} border-l-[0.2vw] bg-yellow-600/30 border-yellow-400`}>
                        <div class="pr-[0.75vw] text-yellow-600 w-[3vw]">
                            <InfoSVG />
                        </div>
                        <div>
                            <p class={`${textClass} text-yellow-400`}>Connecting to backend, please wait...</p>
                        </div>
                    </div>
                )}
                {status() === "connected" && (
                    <div class={`${baseClass} border-l-[0.2vw] bg-blue-600/30 border-blue-400`}>
                        <div class="pr-[0.75vw] text-blue-600 w-[3vw]">
                            <InfoSVG />
                        </div>
                        <div>
                            <p class={`${textClass} text-blue-400`}>Any files you upload will show up here, click on the &apos;Upload&apos; button to start uploading files or Drag & Drop files anywhere on this website</p>
                        </div>
                    </div>
                )}
                {(status() === "error" || status() === "disconnected") && (
                    <div class={`${baseClass} border-1 bg-red-600/30 border-red-400`}>
                        <div class="pr-[0.75vw] text-red-600 w-[3vw]">
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
            class="w-full flex justify-center flex-wrap space-x-8 space-y-8 overflow-y-scroll py-10 custom-scrollbar"
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
                disabled={info() && info()!.status === 'uploading'}
            >
                <BinSVG />
            </button>
        </div>
    )
}

const CHUNK_SIZE = 7 * 1024 * 1024; // 7MB chunk size
const MAX_CONCURRENT_UPLOADS = 3;

function generateClientToken(): string {
    const chars = "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890";
    const randomChoice = (arr: string, k: number): string => {
        let result = "";
        for (let i = 0; i < k; i++) {
            result += arr[Math.floor(Math.random() * arr.length)];
        }
        return result;
    };
    const part1 = randomChoice(chars, 10);
    const part2 = randomChoice(chars, 20);
    const part3 = String(Math.round(Date.now() / 1000));
    return `${part1}.${part2}.${part3}`;
}

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
): Promise<void> {
    
    const backendUrl = import.meta.env.DEV ? 'http://localhost:8080' : '';
    const file = selectableFile.file;
    const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
    let uploadedChunks = 0;

    for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
        const start = chunkIndex * CHUNK_SIZE;
        const end = Math.min(start + CHUNK_SIZE, file.size);
        const chunk = file.slice(start, end);

        const formData = new FormData();
        formData.append('chunk', chunk, file.name);
        formData.append('chunkIndex', String(chunkIndex));

        const response = await fetch(`${backendUrl}/upload/${uploadSystemId}`, {
            method: 'POST',
            body: formData,
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Chunk upload failed (${response.status}): ${errorText}`);
        }
        uploadedChunks++;
        updateProgress(Math.round((uploadedChunks / totalChunks) * 100));
    }

    let finalizeFormData = new FormData();
    let md5sum = await getFileMD5(selectableFile.file)
    finalizeFormData.append('totalChunks', String(totalChunks));
    finalizeFormData.append('originalFileName', file.name);
    finalizeFormData.append('md5sum', md5sum);

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


const DesktopPopUp: Component = () => {
    const [selectedFiles, setSelectedFiles] = createSignal<SelectableFile[]>([]);
    const [uploadProgressMap, setUploadProgressMap] = createSignal<Record<string, FileUploadProgressData>>({});
    const [isUploading, setIsUploading] = createSignal(false);

    const handleFileChange = (event: Event) => {
        const input = event.target as HTMLInputElement;
        if (input.files) {
            const newFiles = Array.from(input.files).map(f => ({
                uniqueId: crypto.randomUUID(),
                file: f,
            }));
            setSelectedFiles(prev => [...prev, ...newFiles]);
            // Initialize progress for new files
            setUploadProgressMap(prevMap => {
                const updatedMap = {...prevMap};
                newFiles.forEach(sf => {
                    if (!updatedMap[sf.uniqueId]) { // Avoid overwriting existing entries if any
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
        }
        if (input) input.value = ''; // Reset input
    };

    const handleFileDelete = (uniqueIdToDelete: string) => {
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
        }
    };

    const handleUploadButtonClick = async () => {
        let authDetails: AuthDetails = {};
        const storedEmail = localStorage.getItem("email");
        const storedPassword = localStorage.getItem("password");

        if (storedEmail && storedPassword) {
            authDetails = { email: storedEmail, password: storedPassword };
        } else {
            let token = localStorage.getItem("token");
            if (!token) {
                token = generateClientToken();
                localStorage.setItem("token", token);
            }
            authDetails = { token };
        }

        if (!authDetails.token && (!authDetails.email || !authDetails.password)) {
            alert("Critical authentication error. Unable to proceed.");
             // Mark all pending files as error
            const currentFiles = selectedFiles();
            const newProgress: Record<string, FileUploadProgressData> = {};
            currentFiles.forEach(sf => {
                newProgress[sf.uniqueId] = {
                    id: sf.uniqueId,
                    name: sf.file.name,
                    progress: 0,
                    status: 'error',
                    errorMessage: 'Authentication configuration error',
                };
            });
            setUploadProgressMap(prev => ({...prev, ...newProgress}));
            return;
        }

        setIsUploading(true);
        const filesToProcess = selectedFiles().filter(sf => {
            const currentStatus = uploadProgressMap()[sf.uniqueId]?.status;
            return currentStatus === 'pending' || currentStatus === 'error';
        });
        
        // Ensure all files to process are marked as pending if they were in error
        setUploadProgressMap(prev => {
            const updated = { ...prev };
            filesToProcess.forEach(sf => {
                updated[sf.uniqueId] = {
                    ...updated[sf.uniqueId], // Keep existing name, id
                    status: 'pending',
                    progress: 0, // Reset progress for retries
                    errorMessage: undefined,
                };
            });
            return updated;
        });
        
        const queue = [...filesToProcess];
        let activeUploads = 0;

        const processNext = async () => {
            if (activeUploads >= MAX_CONCURRENT_UPLOADS || queue.length === 0) {
                if (activeUploads === 0 && queue.length === 0) {
                    setIsUploading(false);
                }
                return;
            }

            activeUploads++;
            const selectableFile = queue.shift();
            if (!selectableFile) {
                activeUploads--;
                processNext();
                return;
            }

            setUploadProgressMap(prev => ({
                ...prev,
                [selectableFile.uniqueId]: {
                    ...prev[selectableFile.uniqueId],
                    status: 'uploading',
                    progress: 0,
                }
            }));
            
            const backendUploadId = crypto.randomUUID();

            try {
                await uploadFileInChunks(
                    selectableFile,
                    backendUploadId,
                    authDetails,
                    (progress) => {
                        setUploadProgressMap(prev => ({
                            ...prev,
                            [selectableFile.uniqueId]: {
                                ...prev[selectableFile.uniqueId],
                                progress,
                            }
                        }));
                    }
                );
                setUploadProgressMap(prev => ({
                    ...prev,
                    [selectableFile.uniqueId]: {
                        ...prev[selectableFile.uniqueId],
                        status: 'completed',
                        progress: 100,
                    }
                }));
            } catch (error: any) {
                console.error(`Error uploading file ${selectableFile.file.name}:`, error);
                setUploadProgressMap(prev => ({
                    ...prev,
                    [selectableFile.uniqueId]: {
                        ...prev[selectableFile.uniqueId],
                        status: 'error',
                        errorMessage: error.message || 'Upload failed',
                    }
                }));
            } finally {
                activeUploads--;
                processNext();
            }
        };

        for (let i = 0; i < MAX_CONCURRENT_UPLOADS && queue.length > 0; i++) {
            processNext();
        }
        if (queue.length === 0 && activeUploads === 0) { // If no files were queued (e.g. all completed)
            setIsUploading(false);
        }
    };

    const filesPendingOrError = createMemo(() => {
        return selectedFiles().filter(sf => {
            const status = uploadProgressMap()[sf.uniqueId]?.status;
            return status === 'pending' || status === 'error';
        }).length;
    });


    return (
        <Dialog onOpenChange={handleDialogStateChange}>
            <Dialog.Trigger class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-blue-600 hover:bg-blue-800 p-[1vh] rounded-[1vh] font-bold translate-y-[4vh]"><UploadSVG />Upload</Dialog.Trigger>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 z-50 bg-black/50 data-open:animate-in data-open:fade-in-0% data-closed:animate-out data-closed:fade-out-0%" />
                <Dialog.Content class="bg-[#0f0f0f] text-white fixed left-1/2 top-1/2 z-50 min-w-[clamp(320px,80vw,600px)] w-auto max-w-[90vw] -translate-x-1/2 -translate-y-1/2 rounded-lg border-2 border-gray-900 bg-corvu-100 px-6 py-5 data-open:animate-in data-open:fade-in-0% data-open:zoom-in-95% data-open:slide-in-from-top-10% data-closed:animate-out data-closed:fade-out-0% data-closed:zoom-out-95% data-closed:slide-out-to-top-10%">
                    <Dialog.Label class="text-lg font-bold">
                        Upload Files
                    </Dialog.Label>
                    <label for="file-upload" class={`rounded-md min-h-[15vh] flex justify-center items-center cursor-pointer my-[1vh] ${selectedFiles().length === 0 ? 'border-dotted border-2 border-blue-800' : ''}`}>
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
                                        />
                                    )}
                                </For>
                            </div>
                        )}
                        <input id="file-upload" type="file" multiple class="hidden" onChange={handleFileChange} />
                    </label>
                    <Show when={selectedFiles().length > 0 && filesPendingOrError() > 0}>
                        <button
                            class="mt-4 w-full bg-blue-600 hover:bg-blue-800 disabled:bg-gray-600 text-white font-bold py-2 px-4 rounded"
                            onClick={handleUploadButtonClick}
                            disabled={isUploading()}
                        >
                            {isUploading() ? 'Uploading...' : `Upload ${filesPendingOrError()} File(s)`}
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
                    <DesktopPopUp />
                </div>
                {props.Files().length === 0 ? <FilesError /> : <DriveBody Files={props.Files} />}
            </div>
        </DesktopTemplate>
    )
}

const MobileDrive: Component<{Files: Accessor<Array<FileData>>}> = (props) => {
    // Note: Mobile upload button is not yet wired with this new logic.
    // It would require a similar PopUp or a different UI flow.
    return (
        <div class="flex flex-col w-full min-h-screen bg-black">
            <Navbar CurrentPage="Files" Type="mobile"/>
            <div class="h-[5vh]"/>
            <div class="w-full flex justify-between items-center px-[3vw]">
                <p class="text-white font-black text-[4vh]">My Files</p>
                <button class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-blue-600 hover:bg-blue-800 p-[1vh] rounded-[1vh] font-bold translate-y-[4vh]"><UploadSVG/>Upload</button>
            </div>
            <div class="w-full h-full flex justify-center items-center">
                {props.Files().length === 0 ? <FilesError /> : <DriveBody Files={props.Files} />}
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
        if (data.type === "get_user_files_response") {
            if (data.error) {
                toast.error(`Error fetching files: ${data.error}`);
            } else {
                ctx.setFiles(data.data.sort((a: FileData, b: FileData) => b.timestamp - a.timestamp) || []);
            }
        } else if (data.type === "file_update") {
            if (data.data.toggle === true) {
                ctx.setFiles(prev => [data.data.File, ...prev]);
            }
        }
    }

    const sendFilesRequest = (socket: WebSocket) => {
        if (socket.readyState === WebSocket.OPEN) {
            if (!localStorage.getItem("token") && (!localStorage.getItem("email") || !localStorage.getItem("password"))) {
                localStorage.setItem("token", generateClientToken());
                localStorage.removeItem("email");
                localStorage.removeItem("password");
            }
            socket.send(JSON.stringify({
                type: "get_user_files",
                data: {
                    email: localStorage.getItem("email") || "",
                    password: localStorage.getItem("password") || "",
                    token: localStorage.getItem("token") || ""
                }
            }));
        }
    }

    createEffect(() => {
        const socket = getSocket();
        if (!socket) return;

        let hasRun = false;

        const runOnce = () => {
            if (!hasRun) {
                console.log("MyDrive.tsx: Sending request to get user files");
                sendFilesRequest(socket);
                hasRun = true;
            }
        };

        socket.addEventListener("message", messageHandler);
        socket.addEventListener("open", runOnce);

        if (socket.readyState === WebSocket.OPEN) {
            runOnce();
        }

        onCleanup(() => {
            socket.removeEventListener("message", messageHandler);
            socket.removeEventListener("open", runOnce);
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

export default MyDrive