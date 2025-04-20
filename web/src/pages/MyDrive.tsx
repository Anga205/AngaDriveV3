import type { Accessor, Component } from "solid-js"
import { createSignal } from "solid-js"
import { DesktopTemplate } from "../components/Template"
import { InfoSVG, UploadSVG, ErrorSVG, EyeSVG, CopySVG, BinSVG } from "../assets/SvgFiles"
import Navbar from "../components/Navbar";
import { useWebSocket } from "../Websockets";
import { FileData } from "../types/types";

const FilesError: Component = () => {
    const baseClass = "flex items-center p-[1vh] rounded-[1vh] w-full";
    const textClass = "text-[1.5vh]";
    const containerClass = "md:max-w-1/3 flex flex-col items-center space-y-[2vh]";
    const {status} = useWebSocket();

    return (
        <div class="w-full h-full flex justify-center items-center">
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

const FilePreview: Component<{link: string}> = (props) => {
    return (
        <img src={props.link} loading="lazy" class="max-h-full max-w-full pointer-events-none" />
    )
}

const FileCard: Component<{ File: FileData }> = (props) => {
    const cache_link = import.meta.env.VITE_CACHE_LINK || "https://cloud.anga.pro/i/";
    const default_link = import.meta.env.VITE_DEFAULT_LINK || "https://i.anga.pro/i/";
    return (
        <div class="flex flex-col w-80 h-96 bg-gray-950 border-gray-800 rounded-lg hover:scale-105 transform transition-transform duration-300 shadow-lg shadow-gray-900/50">
            <div class="flex items-center overflow-hidden justify-center w-full h-[14%] bg-gray-900 rounded-t-lg">
                <p class="text-white text-2xl font-semibold text-nowrap font-mono">{props.File.original_file_name.length >17
                    ? `${props.File.original_file_name.slice(0,17)}...`
                    : props.File.original_file_name}</p>
            </div>
            <div class="flex justify-center items-center w-full h-1/2 overflow-hidden">
                <FilePreview link={props.File.cached?cache_link+props.File.file_directory:default_link+props.File.file_directory}/>
            </div>
            <div class="flex w-full space-x-2 p-2 text-xs border-b-1 border-gray-800">
                <div class="flex flex-col items-end w-1/2 h-full text-gray-700 font-sans">
                    <p>Type:</p>
                    <p>Uploaded Name:</p>
                    <p>Timestamp:</p>
                    <p>Size:</p>
                </div>
                <div class="flex flex-col items-start w-1/2 h-full text-gray-700 font-sans">
                    <p>
                        {(() => {
                            const ext = props.File.file_directory.split('.').pop()?.toLowerCase();
                            if (!ext) return "Unknown";
                            if (["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"].includes(ext)) return "Image";
                            if (["mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"].includes(ext)) return "Video";
                            if (["mp3", "wav", "aac", "flac", "ogg", "wma", "m4a"].includes(ext)) return "Audio";
                            if (["pdf"].includes(ext)) return "Document";
                            if (["doc", "docx", "odt", "rtf", "txt", "md", "tex"].includes(ext)) return "Text Document";
                            if (["xls", "xlsx", "ods", "csv", "tsv"].includes(ext)) return "Spreadsheet";
                            if (["ppt", "pptx", "odp", "key"].includes(ext)) return "Presentation";
                            if (["zip", "rar", "7z", "tar", "gz", "bz2", "xz"].includes(ext)) return "Archive";
                            if (["exe", "msi", "bat", "sh", "apk", "dmg"].includes(ext)) return "Executable";
                            if (["html", "htm", "css", "js", "ts", "jsx", "tsx", "json", "xml", "yaml", "yml"].includes(ext)) return "Code";
                            if (["iso", "img", "bin", "cue"].includes(ext)) return "Disk Image";
                            if (["epub", "mobi", "azw", "azw3"].includes(ext)) return "Ebook";
                            return "Unknown";
                        })()}
                    </p>
                    <p>{props.File.file_directory}</p>
                    <p>{new Date(props.File.timestamp).toLocaleString('en-GB', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', hour12: true })}</p>
                    <p>
                        {(() => {
                            const bytes = props.File.file_size;
                            if (bytes < 1024) return `${bytes} B`;
                            const kb = bytes / 1024;
                            if (kb < 1024) return `${kb.toFixed(2)} KB`;
                            const mb = kb / 1024;
                            if (mb < 1024) return `${mb.toFixed(2)} MB`;
                            const gb = mb / 1024;
                            if (gb < 1024) return `${gb.toFixed(2)} GB`;
                            const tb = gb / 1024;
                            return `${tb.toFixed(2)} TB`;
                        })()}
                    </p>
                </div>
            </div>
            <div class="w-full flex justify-between p-2">
                <div/>
                <button class="flex items-center justify-center p-2 bg-yellow-700/30 hover:bg-yellow-700/20 rounded-xl text-yellow-600">
                    <EyeSVG />
                </button>
                <div/>
                <button class="flex items-center justify-center p-2 bg-cyan-700/30 hover:bg-cyan-700/20 rounded-xl text-cyan-500">
                    <CopySVG />
                </button>
                <div/>
                <button class="flex items-center justify-center p-2 text-red-700 bg-red-800/30 hover:bg-red-900/20 rounded-xl">
                    <BinSVG />
                </button>
                <div/>
            </div>
        </div>
    );
};

const DriveBody: Component<{ Files: Accessor<Array<FileData>> }> = (props) => {
    const sampleFile: FileData = {
        original_file_name: "Sample File.txt",
        file_directory: "0dskxfu5ufmc.jpg",
        file_size: 1024,
        timestamp: Date.now(),
        cached: false,
    };
    return (
        <div 
            class="w-full flex justify-center flex-wrap space-x-8 space-y-8 overflow-y-scroll py-10 custom-scrollbar"
        >
            {props.Files().length === 0 ? (
            Array.from({ length: 15 }).map((_, index) => (
            <FileCard File={{ ...sampleFile, original_file_name: `Sample File ${index + 1}.txt` }} />
            ))
            ) : (
            props.Files().map((file) => <FileCard File={file} />)
            )}
        </div>
    );
};

const DesktopDrive: Component<{Files: Accessor<Array<FileData>>}> = (props) => {
    return (
        <DesktopTemplate CurrentPage="Files">
            <div class="flex flex-col w-full h-full px-[2vh] p-[1vh] space-y-10">
                <div class="w-full flex justify-between items-center">
                    <p class="text-white font-black text-[4vh]">My Files</p>
                    <button class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-blue-600 hover:bg-blue-800 p-[1vh] rounded-[1vh] font-bold translate-y-[4vh]"><UploadSVG/>Upload</button>
                </div>
                {props.Files().length === 1 ? <FilesError /> : <DriveBody Files={props.Files} />}
            </div>
        </DesktopTemplate>
    )
}

const MobileDrive: Component<{Files: Accessor<Array<FileData>>}> = () => {
    return (
        <div class="flex flex-col w-full min-h-screen h-screen bg-black">
            <Navbar CurrentPage="Files" Type="mobile"/>
            <div class="h-[5vh]"/>
            <div class="w-full flex justify-between items-center px-[3vw]">
                <p class="text-white font-black text-[4vh]">My Files</p>
                <button class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-blue-600 hover:bg-blue-800 p-[1vh] rounded-[1vh] font-bold translate-y-[4vh]"><UploadSVG/>Upload</button>
            </div>
            <div class="w-full h-full flex justify-center items-center">
                <FilesError />
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

    const [files] = createSignal<Array<FileData>>([]);

    return (
        <>
            <title>My Files | DriveV3</title>
            {isMobile() ? <MobileDrive Files={files}/> : <DesktopDrive Files={files}/>}
        </>
    )
}

export default MyDrive