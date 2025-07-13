import { Component } from "solid-js";
import type { FileData } from "../library/types"
import { BinSVG, CopySVG, CrossSVG, DownloadSVG, EyeSVG, FileTextSVG, RefreshSVG } from "../assets/SvgFiles";
import { formatFileSize, getFileType } from "../library/functions";
import toast from "solid-toast";
import { useWebSocket } from "../Websockets";
import { useLocation } from "@solidjs/router";

const FilePreview: Component<{ file: FileData }> = (props) => {
    let link = import.meta.env.DEV ? "http://localhost:8080/i/" : import.meta.env.VITE_DEFAULT_LINK || `${window.location.protocol}//${window.location.host}/i/`;
    const preview_size_limit = 50 * 1024 * 1024;
    link += props.file.file_directory;

    const ext = props.file.original_file_name.split('.').pop()?.toLowerCase();

    let previewContent;
    if (props.file.file_size > preview_size_limit) {
        previewContent = (
            <FileTextSVG class="max-h-full p-4 opacity-50"/>
        );
    } else if (!ext) {
        previewContent = <p class="text-white">Unsupported file type</p>;
    } else if (["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"].includes(ext)) {
        previewContent = <img src={link} loading="lazy" class="max-h-full max-w-full p-2" />;
    } else if (["mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"].includes(ext)) {
        previewContent = <video src={link} controls class="max-h-full max-w-full" preload="metadata" />;
    } else if (["mp3", "wav", "aac", "flac", "ogg", "wma", "m4a"].includes(ext)) {
        previewContent = <audio src={link} controls class="w-full" />;
    } else if (["pdf"].includes(ext)) {
        link = import.meta.env.DEV ? "http://localhost:8080/preview/" : import.meta.env.VITE_DEFAULT_LINK || `${window.location.protocol}//${window.location.host}/preview/`;
        link += props.file.file_directory;
        link += '.png'
        previewContent = <img src={link} loading="lazy" class="max-h-full max-w-full p-2" />;
    } else {
        previewContent = (
            <FileTextSVG class="max-h-full p-4 opacity-50"/>
        );
    }

    return <div class="flex justify-center items-center w-full h-full opacity-70">{previewContent}</div>;
};

const ConvertButton: Component<{ file: FileData }> = (props) => {
    const { socket: getSocket } = useWebSocket();
    const handleConvert = async () => {
        const convertRequest = {
            type: "convert_video",
            data: {
                file_directory: props.file.file_directory,
                auth: {
                    token: localStorage.getItem("token") || "",
                    email: localStorage.getItem("email") || "",
                    password: localStorage.getItem("password") || ""
                }
            }
        }
        if (getSocket()?.readyState !== WebSocket.OPEN) {
            toast.error("WebSocket is not available");
            return;
        }
        getSocket()?.send(JSON.stringify(convertRequest));
    };

    return (
        ["mkv", "avi", "mov", "wmv", "flv", "webm"].includes(props.file.original_file_name.split('.').pop()?.toLowerCase() || '') ?
        <>
            <div/>
            <button class="flex items-center justify-center p-2 bg-blue-700/30 hover:bg-blue-700/20 rounded-xl text-blue-500" onClick={handleConvert}>
                <RefreshSVG/>
            </button>
            <div/>
        </>
        : <div/>
    );
}

const DeleteButton: Component<{ file: FileData }> = (props) => {
    const { socket: getSocket } = useWebSocket();
    const handleDelete = async () => {
        const deleteRequest = {
            type: "delete_file",
            data: {
                file_directory: props.file.file_directory,
                auth: {
                    token: localStorage.getItem("token") || "",
                    email: localStorage.getItem("email") || "",
                    password: localStorage.getItem("password") || ""
                }
            }
        }
        if (getSocket()?.readyState !== WebSocket.OPEN) {
            toast.error("WebSocket is not available");
            return;
        }
        getSocket()?.send(JSON.stringify(deleteRequest));
    }
    return (
        <button class="flex items-center justify-center p-2 text-red-700 bg-red-800/30 hover:bg-red-900/20 rounded-xl" onClick={handleDelete}>
            <BinSVG />
        </button>
    )
}

const RemoveFromCollectionButton: Component<{ file: FileData }> = (props) => {
    const { socket: getSocket } = useWebSocket();
    const collectionIdParam = new URLSearchParams(location.search).get("id") || "";
    const ids = collectionIdParam.split(" ");
    const collectionId = ids[ids.length - 1];
    const handleRemove = async () => {
        const removeRequest = {
            type: "remove_file_from_collection",
            data: {
                file_directory: props.file.file_directory,
                collection_id: collectionId,
                auth: {
                    token: localStorage.getItem("token") || "",
                    email: localStorage.getItem("email") || "",
                    password: localStorage.getItem("password") || ""
                }
            }
        }
        if (getSocket()?.readyState !== WebSocket.OPEN) {
            toast.error("WebSocket is not available");
            return;
        }
        getSocket()?.send(JSON.stringify(removeRequest));
    }
    return (
        <button class="flex items-center justify-center p-2 text-red-700 bg-red-800/30 hover:bg-red-900/20 rounded-xl" onClick={handleRemove}>
            <CrossSVG />
        </button>
    )
}

const FileCard: Component<{ File: FileData }> = (props) => {
    let link = import.meta.env.DEV ? "http://localhost:8080/i/" : import.meta.env.VITE_DEFAULT_LINK || `${window.location.protocol}//${window.location.host}/i/`;
    link += props.File.file_directory;
    link = link.split('.').slice(0, -1).join('.');
    link += "/"+props.File.original_file_name;
    const location = useLocation();
    return (
        <div class="flex flex-col w-80 h-96 bg-neutral-950 border-neutral-800 border-1 rounded-lg">
            <div class="flex items-center overflow-hidden justify-center w-full h-[14%] bg-neutral-900 rounded-t-lg">
                <p class="text-white text-2xl font-semibold text-nowrap font-sans">{props.File.original_file_name.length >17
                    ? `${props.File.original_file_name.slice(0,17)}...`
                    : props.File.original_file_name}</p>
            </div>
            <div class="flex justify-center items-center w-full h-1/2 overflow-hidden">
                <FilePreview file={props.File}/>
            </div>
            <div class="flex w-full space-x-2 p-2 text-xs border-b-1 border-neutral-800">
                <div class="flex flex-col items-end w-1/2 h-full text-neutral-700 font-sans">
                    <p>Type:</p>
                    <p>Uploaded Name:</p>
                    <p>Timestamp:</p>
                    <p>Size:</p>
                </div>
                <div class="flex flex-col items-start w-1/2 h-full text-neutral-700 font-sans">
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
                <a class="flex items-center justify-center p-2 bg-yellow-700/30 hover:bg-yellow-700/20 rounded-xl text-yellow-600" href={link} target="_blank">
                    <EyeSVG />
                </a>
                <div/>
                <button class="flex items-center justify-center p-2 bg-cyan-700/30 hover:bg-cyan-700/20 rounded-xl text-cyan-500" onClick={() => {
                    navigator.clipboard.writeText(link)
                    toast.success("Link to "+ props.File.original_file_name + " copied to clipboard!", {
                        duration: 2000,
                        position: "bottom-right",
                        style: {
                            background: "#1f2937",
                            color: "#f3f4f6"
                        }
                    });
                }}>
                    <CopySVG />
                </button>
                <div/>
                <button class="flex items-center justify-center p-2 bg-green-700/30 hover:bg-green-700/20 rounded-xl text-green-500" onClick={async () => {
                    try {
                        const response = await fetch(link, { mode: "cors" });
                        if (!response.ok) {
                            console.error("Failed to fetch file:", response.statusText);
                            return;
                        }
                        const blob = await response.blob();
                        const blobUrl = window.URL.createObjectURL(blob);
                        const downloadLink = document.createElement("a");
                        downloadLink.href = blobUrl;
                        downloadLink.download = props.File.original_file_name;
                        document.body.appendChild(downloadLink);
                        downloadLink.click();
                        document.body.removeChild(downloadLink);
                        window.URL.revokeObjectURL(blobUrl);
                    } catch (error) {
                        console.error("Error downloading file:", error);
                    }
                }}>
                    <DownloadSVG />
                </button>
                <ConvertButton file={props.File} />
                {location.pathname === "/my_drive" ? <DeleteButton file={props.File} /> : <RemoveFromCollectionButton file={props.File} />}
                <div/>
            </div>
        </div>
    );
};

export default FileCard;