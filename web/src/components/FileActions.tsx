import type { FileData } from "../library/types"
import { BinSVG, CrossSVG } from "../assets/SvgFiles";
import { getCollectionPathIds } from "../library/functions";
import toast from "solid-toast";
import { useWebSocket } from "../Websockets";
import { useLocation } from "@solidjs/router";
import { AppContext } from "../Context";
import { createSignal, createMemo, Component, Show, useContext } from "solid-js";
import Dialog from '@corvu/dialog';
import Tooltip from "@corvu/tooltip";
import RotateCcw from "lucide-solid/icons/rotate-ccw";
import Image from "lucide-solid/icons/image";
import { useMenuContext } from "./ContextMenu";

const VIDEO_EXTENSIONS = ["mkv", "avi", "mov", "wmv", "flv", "webm"];
const IMAGE_CONVERT_EXTENSIONS = ["gif", "bmp", "webp", "tiff", "heic", "heif"];

const getExtension = (fileName: string): string => fileName.split('.').pop()?.toLowerCase() || '';

const getAuth = () => ({
    token: localStorage.getItem("token") || "",
    email: localStorage.getItem("email") || "",
    password: localStorage.getItem("password") || ""
});

const ConvertButton: Component<{ file: FileData; onConvert?: () => void }> = (props) => {
    const { socket: getSocket } = useWebSocket();
    const menu = useMenuContext();
    const handleConvert = async () => {
        const convertRequest = {
            type: "convert_video",
            data: {
                file_directory: props.file.file_directory,
                auth: getAuth()
            }
        }
        if (getSocket()?.readyState !== WebSocket.OPEN) {
            toast.error("WebSocket is not available");
            return;
        }
        getSocket()?.send(JSON.stringify(convertRequest));
        props.onConvert?.();
        menu?.close();
        toast.success("Conversion started for " + props.file.original_file_name)
    };

    return (
        VIDEO_EXTENSIONS.includes(getExtension(props.file.original_file_name)) ?
            <button class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-neutral-100 hover:bg-neutral-700" onClick={handleConvert}>
                <RotateCcw class="h-4 w-4 text-neutral-100" />
                <span>Convert to MP4</span>
            </button>
            : <div />
    );
}

const ConvertImageButton: Component<{ file: FileData; target: "png" | "jpg"; label?: string; onConvert?: () => void }> = (props) => {
    const { socket: getSocket } = useWebSocket();
    const handleConvert = async () => {
        const convertRequest = {
            type: "convert_image",
            data: {
                file_directory: props.file.file_directory,
                target_format: props.target,
                auth: getAuth()
            }
        }
        if (getSocket()?.readyState !== WebSocket.OPEN) {
            toast.error("WebSocket is not available");
            return;
        }
        getSocket()?.send(JSON.stringify(convertRequest));
        props.onConvert?.();
        toast.success("Conversion started for " + props.file.original_file_name)
    };

    return (
        <button class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm text-neutral-100 hover:bg-neutral-700" onClick={handleConvert}>
            <Image class="h-4 w-4 text-neutral-100" />
            <span>{props.label || `Convert to ${props.target.toUpperCase()}`}</span>
        </button>
    );
}

const DeleteButton: Component<{ file: FileData }> = (props) => {
    const { socket: getSocket } = useWebSocket();
    const [open, setOpen] = createSignal(false);
    const handleDelete = async () => {
        const deleteRequest = {
            type: "delete_file",
            data: {
                file_directory: props.file.file_directory,
                auth: getAuth()
            }
        }
        if (getSocket()?.readyState !== WebSocket.OPEN) {
            toast.error("WebSocket is not available");
            return;
        }
        getSocket()?.send(JSON.stringify(deleteRequest));
        setOpen(false);
    }
    const handleTriggerClick = (e: MouseEvent) => {
        if (e.shiftKey) {
            e.preventDefault();
            e.stopPropagation();
            handleDelete();
        }
    }
    return (
        <Dialog open={open()} onOpenChange={setOpen}>
            <Tooltip placement="bottom" openDelay={0} closeDelay={0}>
                <Tooltip.Trigger
                    as={Dialog.Trigger}
                    class="flex items-center justify-center p-2 text-red-700 bg-red-800/30 hover:bg-red-900/20 rounded-xl"
                    onClick={handleTriggerClick}
                    aria-label="Delete file"
                >
                    <BinSVG />
                </Tooltip.Trigger>
                <Tooltip.Content class="bg-neutral-900 text-white px-2 py-1 rounded">Delete&nbsp;File</Tooltip.Content>
            </Tooltip>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
                <Dialog.Content class="flex z-50 justify-center flex-col fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-neutral-800 p-6 rounded-md shadow-lg text-white w-[clamp(300px,50vw,500px)]">
                    <Dialog.Label class="text-xl font-semibold mb-2 text-center">
                        Delete {props.file.original_file_name.length > 17
                            ? `${props.file.original_file_name.slice(0, 17)}...`
                            : props.file.original_file_name}?
                    </Dialog.Label>
                    <p class="mb-4 text-sm text-neutral-400 text-center">
                        Once a file is deleted, it may not be recoverable again. Are you sure you want to permanently delete this file?
                    </p>
                    <div class="flex justify-between space-x-3 mt-6">
                        <Dialog.Close class="bg-neutral-600 hover:bg-neutral-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200">
                            Cancel
                        </Dialog.Close>
                        <button
                            class="bg-red-600 hover:bg-red-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200"
                            onClick={handleDelete}
                        >
                            Delete
                        </button>
                    </div>
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    )
}

const RemoveFromCollectionButton: Component<{ file: FileData }> = (props) => {
    const { socket: getSocket } = useWebSocket();
    const ctx = useContext(AppContext)!;
    const location = useLocation();
    // Get the current collection ID from the path (last segment)
    const pathIds = createMemo(() => getCollectionPathIds(location.pathname));
    const collectionId = () => pathIds()[pathIds().length - 1] || "";
    const canRemove = () => {
        const id = collectionId();
        return location.pathname.startsWith("/collection") && !!id && (ctx.knownCollections()[id]?.isOwned || false);
    };
    const handleRemove = async () => {
        const removeRequest = {
            type: "remove_file_from_collection",
            data: {
                file_directory: props.file.file_directory,
                collection_id: collectionId(),
                auth: getAuth()
            }
        }
        if (getSocket()?.readyState !== WebSocket.OPEN) {
            toast.error("WebSocket is not available");
            return;
        }
        getSocket()?.send(JSON.stringify(removeRequest));
        toast.success("Removed from collection", {
            duration: 2000,
            position: "bottom-right",
            style: {
                background: "#1f1f1f",
                color: "#ffffff"
            }
        });
    }
    return (
        <Show when={canRemove()}>
            <Tooltip placement="bottom" openDelay={0} closeDelay={0}>
                <Tooltip.Trigger
                    class="flex items-center justify-center p-2 text-red-700 bg-red-800/30 hover:bg-red-900/20 rounded-xl"
                    onClick={handleRemove}
                    aria-label="Remove from collection"
                >
                    <CrossSVG />
                </Tooltip.Trigger>
                <Tooltip.Content class="bg-neutral-900 text-white px-2 py-1 rounded">
                    Remove&nbsp;From&nbsp;Collection
                </Tooltip.Content>
            </Tooltip>
        </Show>
    )
}

export { ConvertButton, ConvertImageButton, DeleteButton, RemoveFromCollectionButton, VIDEO_EXTENSIONS, IMAGE_CONVERT_EXTENSIONS, getExtension };