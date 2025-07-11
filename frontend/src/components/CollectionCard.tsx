import { Component, onMount, useContext } from "solid-js";
import { BinSVG, CopySVG, CrossSVG, EyeSVG } from "../assets/SvgFiles";
import { CollectionCardData } from "../library/types";
import { formatFileSize, getCollection } from "../library/functions";
import { useWebSocket } from "../Websockets";
import toast from "solid-toast";
import { useNavigate, useLocation } from "@solidjs/router";
import { AppContext } from "../Context";
import Tooltip from "@corvu/tooltip";

const CollectionCard: Component<{ collection: CollectionCardData }> = (props) => {
    const {socket, status} = useWebSocket();
    const location = useLocation();
    const navigate = useNavigate();
    const ctx = useContext(AppContext)!;

    const handleDelete = () => {
        if (status() !== "connected") {
            toast.error("Could not secure a connection to the server. Please try again later.");
            return;
        }
        const ws = socket();
        if (!ws) return;
        ws.send(JSON.stringify({
            type: "delete_collection",
            data: {
                collection_id: props.collection.id,
                auth: {
                    token: localStorage.getItem('token') || '',
                    email: localStorage.getItem('email') || '',
                    password: localStorage.getItem('password') || ''
                }
            }
        }));
    };

    const handleRemove = () => {
        if (status() !== "connected") {
            toast.error("Could not secure a connection to the server. Please try again later.");
            return;
        }
        const ws = socket();
        if (!ws) return;
        const collectionId = new URLSearchParams(location.search).get("id") || "";
        const ids = collectionId.split(" ");
        const lastId = ids[ids.length - 1];

        ws.send(JSON.stringify({
            type: "remove_folder_from_collection",
            data: {
            collection_id: lastId,
            folder_id: props.collection.id,
            auth: {
                token: localStorage.getItem('token') || '',
                email: localStorage.getItem('email') || '',
                password: localStorage.getItem('password') || ''
            }
            }
        }));
    };

    const viewCollection = () => {
        const currentUrl = window.location.href;
        let newUrl = "";

        if (currentUrl.includes("/my_collection")) {
            newUrl = `/collection/?id=${props.collection.id}`;
        } else if (currentUrl.includes("/collection/?id=")) {
            const urlParams = new URLSearchParams(window.location.search);
            const id = urlParams.get('id');
            if (id) {
                const ids = id.split(' ');
                if (ids.length >= 2 && ids[ids.length - 2] === props.collection.id) {
                    const newIds = ids.slice(0, -1);
                    newUrl = `/collection/?id=${newIds.join(' ')}`;
                } else {
                    newUrl = `/collection/?id=${id} ${props.collection.id}`;
                }
            } else {
                newUrl = `/collection/?id=${props.collection.id}`;
            }
        } else {
            newUrl = `/collection/?id=${props.collection.id}`;
        }

        navigate(newUrl);
    };

    const handleCopy = () => {
        const collectionUrl = `${window.location.origin}/collection/?id=${props.collection.id}`;
        navigator.clipboard.writeText(collectionUrl)
            .then(() => {
                toast.success("Collection URL copied to clipboard!");
            })
            .catch(() => {
                toast.error("Failed to copy collection URL.");
            });
    };

    onMount(() => {
        getCollection(props.collection.id, status, socket, ctx);
    });

    return (
        <div class="flex flex-col w-64 h-60 bg-neutral-800 rounded-lg px-4 py-2 pb-3">
            <p class="text-white font-bold text-2xl text-center w-full p-2">{props.collection.name}</p>
            <hr class="border-neutral-600"/>
            <div class="flex w-full items-center justify-center h-full my-2">
                <div class="flex flex-col items-start justify-between h-full">
                    <p class="text-gray-400 text-sm">Size:</p>
                    <p class="text-gray-400 text-sm">File Count:</p>
                    <p class="text-gray-400 text-sm">Folder Count:</p>
                    <p class="text-gray-400 text-sm">Editors:</p>
                </div>
                <div class="flex flex-col items-start justify-between h-full ml-4">
                    <p class="text-white text-sm">{formatFileSize(props.collection.size)}</p>
                    <p class="text-white text-sm">{props.collection.file_count}</p>
                    <p class="text-white text-sm">{props.collection.folder_count}</p>
                    <p class="text-white text-sm">{props.collection.editor_count}</p>
                </div>
            </div>
            <div class="flex w-full justify-between px-5">
                <Tooltip placement="bottom" openDelay={0} closeDelay={0}>
                    <Tooltip.Trigger
                        onClick={viewCollection}
                        class="flex items-center justify-center p-2 bg-yellow-700/30 hover:bg-yellow-700/20 rounded-xl text-yellow-600"
                    >
                        <EyeSVG />
                    </Tooltip.Trigger>
                    <Tooltip.Content class="bg-neutral-900 text-white px-2 py-1 rounded">
                        View Collection
                    </Tooltip.Content>
                </Tooltip>
                <Tooltip placement="bottom" openDelay={0} closeDelay={0}>
                    <Tooltip.Trigger
                        onClick={handleCopy}
                        class="flex items-center justify-center p-2 bg-lime-700/30 hover:bg-lime-700/20 rounded-xl text-lime-600"
                    >
                        <CopySVG />
                    </Tooltip.Trigger>
                    <Tooltip.Content class="bg-neutral-900 text-white px-2 py-1 rounded">
                        Copy Link to Collection
                    </Tooltip.Content>
                </Tooltip>
                {location.pathname.startsWith("/collection") ? (
                    <Tooltip placement="bottom" openDelay={0} closeDelay={0}>
                        <Tooltip.Trigger
                            onClick={handleRemove}
                            class="flex items-center justify-center p-2 bg-red-700/30 hover:bg-red-700/20 rounded-xl text-red-600"
                        >
                            <CrossSVG />
                        </Tooltip.Trigger>
                        <Tooltip.Content class="bg-neutral-900 text-white px-2 py-1 rounded">
                            Remove From Collection
                        </Tooltip.Content>
                    </Tooltip>
                ) : (
                    <Tooltip placement="bottom" openDelay={0} closeDelay={0}>
                        <Tooltip.Trigger
                            onClick={handleDelete}
                            class="flex items-center justify-center p-2 bg-red-700/30 hover:bg-red-700/20 rounded-xl text-red-600"
                        >
                            <BinSVG />
                        </Tooltip.Trigger>
                        <Tooltip.Content class="bg-neutral-900 text-white px-2 py-1 rounded">
                            Delete Collection
                        </Tooltip.Content>
                    </Tooltip>
                )}
            </div>
        </div>
    )
}
export default CollectionCard;