import { Component } from "solid-js";
import type { FileData } from "../types/types"
import { BinSVG, CopySVG, DownloadSVG, EyeSVG, FileTextSVG } from "../assets/SvgFiles";

const FilePreview: Component<{ file: FileData }> = (props) => {
    const cache_link = import.meta.env.DEV ? "http://localhost:8080/i/" : import.meta.env.VITE_CACHE_LINK || "https://cloud.anga.pro/i/";
    const default_link = import.meta.env.DEV ? "http://localhost:8080/i/" : import.meta.env.VITE_DEFAULT_LINK || "https://i.anga.pro/i/";
    const preview_size_limit = 50 * 1024 * 1024;

    const link = props.file.cached ? cache_link + props.file.file_directory : default_link + props.file.file_directory;

    const ext = link.split('.').pop()?.toLowerCase();

    let previewContent;
    if (props.file.file_size > preview_size_limit) {
        previewContent = (
            <FileTextSVG class="max-h-full p-4 opacity-50"/>
        );
    } else if (!ext) {
        previewContent = <p class="text-white">Unsupported file type</p>;
    } else if (["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"].includes(ext)) {
        previewContent = <img src={link} loading="lazy" class="max-h-full max-w-full" />;
    } else if (["mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"].includes(ext)) {
        previewContent = <video src={link} controls class="max-h-full max-w-full" preload="metadata" />;
    } else if (["mp3", "wav", "aac", "flac", "ogg", "wma", "m4a"].includes(ext)) {
        previewContent = <audio src={link} controls class="w-full" />;
    } else if (["pdf"].includes(ext)) {
        previewContent = (
            <iframe
                src={link}
                class="w-full h-full"
                title="PDF Preview"
                loading="lazy"
            />
        );
    } else {
        previewContent = (
            <FileTextSVG class="max-h-full p-4 opacity-50"/>
        );
    }

    return <div class="flex justify-center items-center w-full h-full opacity-70">{previewContent}</div>;
};

const formatFileSize = (size: number) => {
    const units = ["B", "KB", "MB", "GB", "TB"];
    let index = 0;
    let formattedSize = size;

    while (formattedSize >= 1024 && index < units.length - 1) {
        formattedSize /= 1024;
        index++;
    }

    return `${formattedSize.toFixed(2)} ${units[index]}`;
};

const getFileType = (filePath: string) => {
    const ext = filePath.split('.').pop()?.toLowerCase();
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
    if (["html", "htm", "css", "js", "ts", "jsx", "tsx", "json", "xml", "yaml", "yml", "c", "py"].includes(ext)) return "Code";
    if (["iso", "img", "bin", "cue"].includes(ext)) return "Disk Image";
    if (["epub", "mobi", "azw", "azw3"].includes(ext)) return "Ebook";
    return "Unknown";
};

const FileCard: Component<{ File: FileData }> = (props) => {
    return (
        <div class="flex flex-col w-80 h-96 bg-gray-950 border-gray-800 rounded-lg">
            <div class="flex items-center overflow-hidden justify-center w-full h-[14%] bg-gray-900 rounded-t-lg">
                <p class="text-white text-2xl font-semibold text-nowrap font-mono">{props.File.original_file_name.length >17
                    ? `${props.File.original_file_name.slice(0,17)}...`
                    : props.File.original_file_name}</p>
            </div>
            <div class="flex justify-center items-center w-full h-1/2 overflow-hidden">
                <FilePreview file={props.File}/>
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
                        {getFileType(props.File.file_directory)}
                    </p>
                    <p>{props.File.file_directory}</p>
                    <p>{new Date(props.File.timestamp*1000).toLocaleString('en-GB', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', hour12: true })}</p>
                    <p>
                        {formatFileSize(props.File.file_size)}
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
                <button class="flex items-center justify-center p-2 bg-green-700/30 hover:bg-green-700/20 rounded-xl text-green-500">
                    <DownloadSVG />
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

export default FileCard;