import type { Accessor, Component } from "solid-js"
import { createSignal } from "solid-js"
import { DesktopTemplate } from "../components/Template"
import { InfoSVG, UploadSVG, ErrorSVG, BinSVG, FileSVG } from "../assets/SvgFiles"
import Navbar from "../components/Navbar";
import { useWebSocket } from "../Websockets";
import { FileData } from "../types/types";
import Dialog from "@corvu/dialog";
import FileCard from "../components/FileCard";

const FilesError: Component = () => {
    const baseClass = "flex items-center p-[1vh] rounded-[1vh] w-full";
    const textClass = "text-[1.5vh]";
    const containerClass = "md:max-w-1/3 flex flex-col items-center space-y-[2vh]";
    const {status} = useWebSocket();

    return (
        <div class="w-full h-full flex justify-center items-center px-10 md:px-0">
            <div class={containerClass}>
                {status() === "connecting" && (
                    <div class={`${baseClass} border-l-4 bg-yellow-600/30 border-yellow-400`}>
                        <div class="pr-[1.5vh] text-yellow-600">
                            <InfoSVG />
                        </div>
                        <div>
                            <p class={`${textClass} text-yellow-400`}>Connecting to backend, please wait...</p>
                        </div>
                    </div>
                )}
                {status() === "connected" && (
                    <div class={`${baseClass} border-l-4 bg-blue-600/30 border-blue-400`}>
                        <div class="pr-[1.5vh] text-blue-600">
                            <InfoSVG />
                        </div>
                        <div>
                            <p class={`${textClass} text-blue-400`}>Any files you upload will show up here, click on the &apos;Upload&apos; button to start uploading files or Drag & Drop files anywhere on this website</p>
                        </div>
                    </div>
                )}
                {(status() === "error" || status() === "disconnected") && (
                    <div class={`${baseClass} border-1 bg-red-600/30 border-red-400`}>
                        <div class="pr-[1.5vh] text-red-600">
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
            {props.Files().map((file) => <FileCard File={file} />)}
        </div>
    );
};

const FileUploadPreview: Component<{ file: File; index: number; handleFileDelete: (index: number) => void; }> = (props) => {

    const formatFileSize = (size: number) => {
        if (size < 1024) return `${size} B`;
        const kb = size / 1024;
        if (kb < 1024) return `${kb.toFixed(2)} KB`;
        const mb = kb / 1024;
        if (mb < 1024) return `${mb.toFixed(2)} MB`;
        const gb = mb / 1024;
        return `${gb.toFixed(2)} GB`;
    };

    const truncateFileName = (name: string) => {
        return name.length > 32 ? `${name.slice(0, 32)}...` : name;
    };

    const preview_size_limit = 100 * 1024 * 1024; // 100 MB
    const ext = props.file.name.split('.').pop()?.toLowerCase();

    return (
        <div class="flex items-center justify-between p-3 bg-gray-900 rounded-lg shadow-lg hover:shadow-xl transition-shadow">
            <div class="flex items-center space-x-4">
                <div class="w-12 h-12 flex items-center justify-center bg-gray-800 rounded-md overflow-hidden">
                    {(() => {
                        if (ext && ["pdf"].includes(ext)) {
                            return <p class="text-white text-xs">PDF</p>;
                        }
                        if (ext && ["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"].includes(ext) && (props.file.size < preview_size_limit)) {
                            return <img src={URL.createObjectURL(props.file)} alt="Preview" class="w-full h-full object-cover" />;
                        }
                        if (ext && ["mp4", "mkv", "mov", "wmv", "flv", "webm"].includes(ext) && (props.file.size < preview_size_limit)) {
                            return <video src={URL.createObjectURL(props.file)} class="w-full h-full object-cover" muted />;
                        }
                        return <FileSVG />;
                    })()}
                </div>
                <div class="flex flex-col">
                    <p class="text-white text-sm font-medium">{truncateFileName(props.file.name)}</p>
                    <p class="text-gray-400 text-xs">{formatFileSize(props.file.size)}</p>
                </div>
            </div>
            <button
                class="flex items-center justify-center p-2 bg-red-700/30 hover:bg-red-700/20 rounded-lg text-red-500 hover:text-red-700 transition-colors"
                onClick={() => props.handleFileDelete(props.index)}
            >
                <BinSVG />
            </button>
        </div>
    )
}

const DesktopPopUp: Component = () => {
    const [selectedFiles, setSelectedFiles] = createSignal<File[]>([]);

    const handleFileChange = (event: Event) => {
        const input = event.target as HTMLInputElement;
        if (input.files) {
            setSelectedFiles(Array.from(input.files));
        }
    };

    const handleFileDelete = (index: number) => {
        setSelectedFiles((prev) => prev.filter((_, i) => i !== index));
    };

    return (
        <Dialog onOpenChange={() => setSelectedFiles([])}>
            <Dialog.Trigger class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-blue-600 hover:bg-blue-800 p-[1vh] rounded-[1vh] font-bold translate-y-[4vh]"><UploadSVG />Upload</Dialog.Trigger>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 z-50 bg-black/50 data-open:animate-in data-open:fade-in-0% data-closed:animate-out data-closed:fade-out-0%" />
                <Dialog.Content class="bg-[#0f0f0f] text-white fixed left-1/2 top-1/2 z-50 min-w-80 -translate-x-1/2 -translate-y-1/2 rounded-lg border-2 border-gray-900 border-corvu-400 bg-corvu-100 px-6 py-5 data-open:animate-in data-open:fade-in-0% data-open:zoom-in-95% data-open:slide-in-from-top-10% data-closed:animate-out data-closed:fade-out-0% data-closed:zoom-out-95% data-closed:slide-out-to-top-10%">
                    <Dialog.Label class="text-lg font-bold">
                        Upload or Import Files
                    </Dialog.Label>
                    <label for="file-upload" class={`rounded-md min-w-[30vw] min-h-[15vh] flex justify-center items-center cursor-pointer my-[1vh] ${selectedFiles().length === 0 ? 'border-dotted border-2 border-blue-800' : ''}`}>
                        {selectedFiles().length === 0 ? (
                            <p>Drag and drop files here or click to select files</p>
                        ) : (
                            <div class="flex flex-col w-full space-y-4 max-h-[30vh] overflow-y-auto custom-scrollbar">
                                {selectedFiles().map((file, index) => <FileUploadPreview file={file} index={index} handleFileDelete={handleFileDelete} />)}
                            </div>
                        )}
                        <input id="file-upload" type="file" multiple class="hidden" onChange={handleFileChange} />
                    </label>
                    {selectedFiles().length > 0 && (
                        <button class="mt-4 w-full bg-blue-600 hover:bg-blue-800 text-white font-bold py-2 px-4 rounded">
                            Upload
                        </button>
                    )}
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

    const sampleFile: FileData = {
        original_file_name: "Sample File.jpg",
        file_directory: "00oswbkqftd9.pdf",
        file_size: 1024,
        timestamp: Date.now(),
        cached: false,
    };

    const [files] = createSignal<Array<FileData>>(
        Array.from({ length: 12 }).map((_, index) => ({
            ...sampleFile,
            original_file_name: `Sample File ${index + 1}.jpg`,
        }))
    );

    return (
        <>
            <title>My Files | DriveV3</title>
            {isMobile() ? <MobileDrive Files={files}/> : <DesktopDrive Files={files}/>}
        </>
    )
}

export default MyDrive