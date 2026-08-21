import type { FileData } from "../library/types"
import { BinSVG, CopySVG, CrossSVG, DownloadSVG, EyeSVG, FileTextSVG, RefreshSVG } from "../assets/SvgFiles";
import { formatFileSize, getFileType } from "../library/functions";
import toast from "solid-toast";
import { useWebSocket } from "../Websockets";
import { useLocation } from "@solidjs/router";
import { AppContext } from "../Context";
import { createSignal, onCleanup, Component, Show, useContext } from "solid-js";
import { assetsUrl } from "@/assets/ApiUrl";

const FilePreview: Component<{ file: FileData }> = (props) => {
    const ctx = useContext(AppContext)!;
    const [isVisible, setIsVisible] = createSignal<boolean>(ctx.loadedFiles?.()?.has(props.file.file_directory) || false);
    let containerRef: HTMLDivElement | undefined;
    let observer: IntersectionObserver | undefined;

    // Find the nearest scrollable ancestor to use as the IntersectionObserver root
    const getScrollParent = (node: HTMLElement | null): HTMLElement | null => {
        let el: HTMLElement | null = node?.parentElement || null;
        while (el) {
            const style = getComputedStyle(el);
            const overflowY = style.overflowY;
            const overflow = style.overflow;
            const isScrollable = [overflowY, overflow].some((v) => v === "auto" || v === "scroll" || v === "overlay");
            if (isScrollable) return el;
            el = el.parentElement;
        }
        return null;
    };

    const isInView = (el: HTMLElement, rootEl: HTMLElement | null): boolean => {
        const rootRect = rootEl ? rootEl.getBoundingClientRect() : document.documentElement.getBoundingClientRect();
        const rect = el.getBoundingClientRect();
        return (
            rect.bottom > rootRect.top &&
            rect.top < rootRect.bottom &&
            rect.right > rootRect.left &&
            rect.left < rootRect.right
        );
    };

    const setRef = (el: HTMLDivElement) => {
        containerRef = el;
        if (!containerRef) return;

        const rootEl = getScrollParent(containerRef);

        // Create observer with the correct root (scroll container or viewport)
        observer = new IntersectionObserver(
            (entries) => {
                const entry = entries[0];
                if (entry.isIntersecting) {
                    setIsVisible(true);
                    // Persist that this file was loaded for the session
                    try {
                        ctx.setLoadedFiles?.((prev) => {
                            const next = new Set(prev || new Set());
                            next.add(props.file.file_directory);
                            return next;
                        });
                    } catch (e) {
                        // ignore if context not available
                    }
                    observer?.unobserve(entry.target as Element);
                }
            },
            { root: rootEl, threshold: 0.01 }
        );

        // Defer observe to the next frame to avoid Chromium initial layout race
        requestAnimationFrame(() => {
            if (!containerRef) return;
            observer?.observe(containerRef);
        });

        // Fallback manual check (Chromium sometimes doesn't fire until scroll in nested scrollers)
        requestAnimationFrame(() => {
            if (!isVisible() && containerRef && isInView(containerRef, rootEl)) {
                setIsVisible(true);
                try {
                    ctx.setLoadedFiles?.((prev) => {
                        const next = new Set(prev || new Set());
                        next.add(props.file.file_directory);
                        return next;
                    });
                } catch (e) { }
                observer?.unobserve(containerRef);
            }
        });
    };

    onCleanup(() => {
        if (containerRef) {
            observer?.unobserve(containerRef);
        }
        observer?.disconnect();
        observer = undefined;
    });

    const PreviewContent: Component = () => {
        let link = assetsUrl(`/i/${props.file.file_directory}`);
        const preview_size_limit = 40 * 1024 * 1024; // 40 MB

        const ext = props.file.original_file_name.split('.').pop()?.toLowerCase();

        if (props.file.file_size > preview_size_limit) {
            return <FileTextSVG class="max-h-full p-4 opacity-50" />;
        }
        if (!ext) {
            return <p class="text-white">Unsupported file type</p>;
        }
        if (["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff", "heic", "heif"].includes(ext)) {
            link = assetsUrl(`/preview-image/${props.file.file_directory}`);
            return <img src={link} loading="lazy" class="max-h-full max-w-full p-2" />;
        }
        if (ext === "svg") {
            if (props.file.file_size > 200 * 1024) {
                return <FileTextSVG class="max-h-full p-4 opacity-50" />;
            }
            link = assetsUrl(`/preview-image/${props.file.file_directory}`);
            return <img src={link} loading="lazy" class="max-h-full max-w-full p-2" />;
        }
        if (["mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"].includes(ext)) {
            return <video src={link} controls class="max-h-full max-w-full" preload="metadata" />;
        }
        if (["mp3", "wav", "aac", "flac", "ogg", "wma", "m4a"].includes(ext)) {
            return <audio src={link} controls class="w-full" />;
        }
        if (["pdf"].includes(ext)) {
            link = assetsUrl(`/preview/${props.file.file_directory}.png`);
            return <img src={link} loading="lazy" class="max-h-full max-w-full p-2" />;
        }
        return <FileTextSVG class="max-h-full p-4 opacity-50" />;
    };

    return (
        <div ref={setRef} class="flex justify-center items-center w-full h-full opacity-70">
            <Show when={isVisible()} fallback={<FileTextSVG class="max-h-full p-4 opacity-50" />}>
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
                <div />
                <button class="flex items-center justify-center p-2 bg-blue-700/30 hover:bg-blue-700/20 rounded-xl text-blue-500" onClick={handleConvert}>
                    <RefreshSVG />
                </button>
                <div />
            </>
            : <div />
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
    let DownloadLink = assetsUrl(`/download/${props.File.file_directory}`);
    let link = assetsUrl(`/i/${props.File.file_directory}`);
    link = link.split('.').slice(0, -1).join('.');
    link += "/" + props.File.original_file_name;
    while (link.includes(" ")) {
        link = link.replace(" ", "%20");
    }
    const location = useLocation();
    return (
        <div class="flex flex-col w-80 h-96 bg-neutral-950 border-neutral-800 border rounded-lg md:hover:scale-105 transition-transform duration-200 shadow-lg">
            <a class="w-full h-[calc(14%+50%+21.4%)]" href={link} target="_blank" rel="noopener noreferrer">
                <div class="flex items-center overflow-hidden justify-center w-full h-[16.393442623%] bg-neutral-900 rounded-t-lg">
                    <p class="text-white text-2xl font-semibold text-nowrap font-sans">{props.File.original_file_name.length > 17
                        ? `${props.File.original_file_name.slice(0, 17)}...`
                        : props.File.original_file_name}</p>
                </div>
                <div class="flex justify-center items-center w-full h-[58.5480093677%] overflow-hidden">
                    <FilePreview file={props.File} />
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
                        <p>{new Date(props.File.timestamp * 1000).toLocaleString('en-GB', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', hour12: true })}</p>
                        <p>
                            {formatFileSize(props.File.file_size)}
                        </p>
                    </div>
                </div>
            </a>
            <div class="w-full flex justify-between p-2 h-[14.6%]">
                <div />
                <a class="flex items-center justify-center p-2 bg-yellow-700/30 hover:bg-yellow-700/20 rounded-xl text-yellow-600" href={link} target="_blank">
                    <EyeSVG />
                </a>
                <div />
                <button class="flex items-center justify-center p-2 bg-cyan-700/30 hover:bg-cyan-700/20 rounded-xl text-cyan-500" onClick={() => {
                    navigator.clipboard.writeText(link)
                    toast.success("Link to " + props.File.original_file_name + " copied to clipboard!", {
                        duration: 2000,
                        position: "bottom-right",
                        style: {
                            background: "#1f2937",
                            color: "#ffffff"
                        }
                    });
                }}>
                    <CopySVG />
                </button>
                <div />
                <button
                    class="flex items-center justify-center p-2 bg-green-700/30 hover:bg-green-700/20 rounded-xl text-green-500 cursor-pointer"
                    onClick={async () => {
                        const filename = props.File.original_file_name;
                        // Reactive progress state so the toast updates in place
                        const [progressText, setProgressText] = createSignal(`Downloading ${filename}...`);
                        const progressToast = toast.custom(() => (
                            <div
                                class="px-4 py-3 rounded-md shadow-md font-medium"
                                style={{
                                    "background-color": "#2a2a2a",
                                    "color": "#ffffff"
                                }}
                            >
                                {progressText()}
                            </div>
                        ), {
                            duration: 99999999,
                            position: "bottom-right"
                        });

                        try {
                            // Fetch with progress tracking
                            const response = await fetch(DownloadLink);
                            if (!response.ok) throw new Error(`HTTP ${response.status}`);

                            const reader = response.body?.getReader();
                            if (!reader) throw new Error("No response body");

                            const contentLength = Number(response.headers.get("Content-Length")) || 0;
                            let receivedLength = 0;
                            const chunks: Uint8Array<ArrayBuffer>[] = [];

                            while (true) {
                                const { done, value } = await reader.read();
                                if (done) break;
                                chunks.push(value);
                                receivedLength += value.length;

                                if (contentLength > 0) {
                                    const percent = Math.round((receivedLength / contentLength) * 100);
                                    setProgressText(`${percent}% - Downloading ${filename}... (${formatFileSize(receivedLength)}/${formatFileSize(contentLength)})`);
                                }
                            }

                            // Create blob and trigger download
                            const blob = new Blob(chunks, { type: response.headers.get("content-type") || "application/octet-stream" });
                            const url = URL.createObjectURL(blob);
                            const anchor = document.createElement("a");
                            anchor.href = url;
                            anchor.download = filename;
                            anchor.rel = "noopener noreferrer";
                            document.body.appendChild(anchor);
                            anchor.click();
                            anchor.remove();
                            URL.revokeObjectURL(url);

                            toast.success(`Downloaded ${filename}`, {
                                duration: 3000,
                                position: "bottom-right",
                                style: {
                                    background: "#2a2a2a",
                                    color: "#ffffff"
                                }
                            });
                        } catch (err) {
                            toast.error(`Download failed: ${err instanceof Error ? err.message : "Unknown error"}`, {
                                duration: 5000,
                                position: "bottom-right",
                                style: {
                                    background: "#2a2a2a",
                                    color: "#ffffff"
                                }
                            });
                        } finally {
                            toast.dismiss(progressToast);
                        }
                    }}
                >
                    <DownloadSVG />
                </button>
                {location.pathname === "/my_drive" ? <ConvertButton file={props.File} /> : <div />}
                {location.pathname === "/my_drive" ? <DeleteButton file={props.File} /> : <RemoveFromCollectionButton file={props.File} />}
                <div />
            </div>
        </div>
    );
};

export default FileCard;