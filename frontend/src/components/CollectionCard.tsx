import { Component, onMount } from "solid-js";
import { BinSVG, CopySVG, EyeSVG } from "../assets/SvgFiles";
import { CollectionCardData } from "../library/types";
import { formatFileSize } from "../library/functions";
import { useWebSocket } from "../Websockets";
import toast from "solid-toast";
import { useNavigate } from "@solidjs/router";

const CollectionCard: Component<{ collection: CollectionCardData }> = (props) => {
    const {socket, status} = useWebSocket();
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
    }

    const navigate = useNavigate();
    const viewCollection = () => {
        const currentUrl = window.location.href;
        let newUrl = "";

        if (currentUrl.includes("/my_collection")) {
            newUrl = `/collection/?id=${props.collection.id}`;
        } else if (currentUrl.includes("/collection/?id=")) {
            const urlParams = new URLSearchParams(window.location.search);
            const id = urlParams.get('id');
            newUrl = `/collection/?id=${id}+${props.collection.id}`;
        } else {
            newUrl = `/collection/?id=${props.collection.id}`;
        }

        navigate(newUrl);
    }
    const handleCopy = () => {
        const collectionUrl = `${window.location.origin}/collection/?id=${props.collection.id}`;
        navigator.clipboard.writeText(collectionUrl)
            .then(() => {
                toast.success("Collection URL copied to clipboard!");
            })
            .catch(() => {
                toast.error("Failed to copy collection URL.");
            })
    }
    const getCollectionInfo = () => {
        if (status() !== "connected") {
            setInterval(getCollectionInfo, 1000);
            return;
        } else {
            const ws = socket()!;
            ws.send(JSON.stringify({
                type: "get_collection",
                data: {
                    id: props.collection.id,
                    auth: {
                        token: localStorage.getItem("token") || "",
                        email: localStorage.getItem("email") || "",
                        password: localStorage.getItem("password") || ""
                    }
                }
            }))
        }
    }
    onMount(() => {
        getCollectionInfo();
    })
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
                    <p class="text-white text-sm text-start">{props.collection.editor_count}</p>
                </div>
            </div>
            <div class="flex w-full justify-between px-5">
                <a onClick={viewCollection} class="flex items-center justify-center p-2 bg-yellow-700/30 hover:bg-yellow-700/20 rounded-xl text-yellow-600">
                    <EyeSVG />
                </a>
                <a onClick={handleCopy} class="flex items-center justify-center p-2 bg-lime-700/30 hover:bg-lime-700/20 rounded-xl text-lime-600">
                    <CopySVG />
                </a>
                <a onClick={handleDelete} class="flex items-center justify-center p-2 bg-red-700/30 hover:bg-red-700/20 rounded-xl text-red-600">
                    <BinSVG />
                </a>
            </div>
        </div>
    )
}
export default CollectionCard;