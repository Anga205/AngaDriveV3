import type { FileData } from "../library/types"
import { BinSVG, CopySVG, CrossSVG, DownloadSVG, EyeSVG, FileTextSVG, RefreshSVG } from "../assets/SvgFiles";
import { formatFileSize, getFileType } from "../library/functions";
import toast from "solid-toast";
import { useWebSocket } from "../Websockets";
import { useLocation } from "@solidjs/router";
import { AppContext } from "../Context";
import { createSignal, onCleanup, Component, Show, useContext } from "solid-js";

const FilePreview: Component<{ file: FileData }> = (props) => {
    const [isVisible, setIsVisible] = createSignal(false);
    let containerRef: HTMLDivElement | undefined;

    const observer = new IntersectionObserver(
        (entries) => {
            const entry = entries[0];
            if (entry.isIntersecting) {
                setIsVisible(true);
                observer.unobserve(entry.target);
            }
        },
        { threshold: 0.1 }
    );

    const setRef = (el: HTMLDivElement) => {
        containerRef = el;
        if (containerRef) {
            observer.observe(containerRef);
        }
    };

    onCleanup(() => {
        if (containerRef) {
            observer.unobserve(containerRef);
        }
        observer.disconnect();
    });

    const PreviewContent: Component = () => {
        const AssetsURL = import.meta.env.VITE_ASSETS_URL ? `${window.location.protocol}//${import.meta.env.VITE_ASSETS_URL}` : "http://localhost:8080";
        let link = import.meta.env.DEV ? "http://localhost:8080/i/" : `${AssetsURL}/i/`;
        const preview_size_limit = 40 * 1024 * 1024;
        link += props.file.file_directory;

        const ext = props.file.original_file_name.split('.').pop()?.toLowerCase();

        if (props.file.file_size > preview_size_limit) {
            return <FileTextSVG class="max-h-full p-4 opacity-50"/>;
        }
        if (!ext) {
            return <p class="text-white">Unsupported file type</p>;
        }
        if (["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"].includes(ext)) {
            link = import.meta.env.DEV ? "http://localhost:8080/preview-image/" : `${AssetsURL}/preview-image/`;
            link += props.file.file_directory;
            return <img src={link} loading="lazy" class="max-h-full max-w-full p-2" />;
        }
        if (["mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"].includes(ext)) {
            return <video src={link} controls class="max-h-full max-w-full" preload="metadata" />;
        }
        if (["mp3", "wav", "aac", "flac", "ogg", "wma", "m4a"].includes(ext)) {
            return <audio src={link} controls class="w-full" />;
        }
        if (["pdf"].includes(ext)) {
            link = import.meta.env.DEV ? "http://localhost:8080/preview/" : `${AssetsURL}/preview/`;
            link += props.file.file_directory;
            link += '.png'
            return <img src={link} loading="lazy" class="max-h-full max-w-full p-2" />;
        }
        return <FileTextSVG class="max-h-full p-4 opacity-50"/>;
    };

    return (
        <div ref={setRef} class="flex justify-center items-center w-full h-full opacity-70">
            <Show when={isVisible()} fallback={<FileTextSVG class="max-h-full p-4 opacity-50"/>}>
                <PreviewContent />
            </Show>
        </div>
    );
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
        toast.success("Conversion started for " + props.file.original_file_name)
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
    const ctx = useContext(AppContext)!;
    const location = useLocation();
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
        ctx.knownCollections()[collectionId]?.isOwned && (
            <button class="flex items-center justify-center p-2 text-red-700 bg-red-800/30 hover:bg-red-900/20 rounded-xl" onClick={handleRemove}>
                <CrossSVG />
            </button>
        )
    )
}

const FileCard: Component<{ File: FileData }> = (props) => {
    let DownloadLink = import.meta.env.VITE_ASSETS_URL ? `${window.location.protocol}//${import.meta.env.VITE_ASSETS_URL}/download/` : "http://localhost:8080/download/";
    DownloadLink += props.File.file_directory;
    const AssetsURL = import.meta.env.VITE_ASSETS_URL ? `${window.location.protocol}//${import.meta.env.VITE_ASSETS_URL}/i/` : "http://localhost:8080/i/";
    let link = import.meta.env.DEV ? "http://localhost:8080/i/" : AssetsURL;
    link += props.File.file_directory;
    link = link.split('.').slice(0, -1).join('.');
    link += "/"+props.File.original_file_name.replace(" ", "%20");
    const location = useLocation();
    return (
        <div class="flex flex-col w-80 h-96 bg-neutral-950 border-neutral-800 border rounded-lg md:hover:scale-105 transition-transform duration-200 shadow-lg">
            <a class="w-full h-[calc(14%+50%+21.4%)]" href={link} target="_blank" rel="noopener noreferrer">
                <div class="flex items-center overflow-hidden justify-center w-full h-[16.393442623%] bg-neutral-900 rounded-t-lg">
                    <p class="text-white text-2xl font-semibold text-nowrap font-sans">{props.File.original_file_name.length >17
                        ? `${props.File.original_file_name.slice(0,17)}...`
                        : props.File.original_file_name}</p>
                </div>
                <div class="flex justify-center items-center w-full h-[58.5480093677%] overflow-hidden">
                    <FilePreview file={props.File}/>
                </div>
                <div class="flex w-full space-x-2 p-2 text-xs border-b border-neutral-800 h-[25.0585480094%]">
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
            </a>
            <div class="w-full flex justify-between p-2 h-[14.6%]">
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
                <a class="flex items-center justify-center p-2 bg-green-700/30 hover:bg-green-700/20 rounded-xl text-green-500" href={DownloadLink} target="_blank">
                    <DownloadSVG />
                </a>
                {location.pathname === "/my_drive" ? <ConvertButton file={props.File} /> : <div/>}
                {location.pathname === "/my_drive" ? <DeleteButton file={props.File} /> : <RemoveFromCollectionButton file={props.File} />}
                <div/>
            </div>
        </div>
    );
};

export default FileCard;