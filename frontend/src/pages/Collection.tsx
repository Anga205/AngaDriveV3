import { Component, For, useContext, createSignal } from "solid-js";
import { DesktopTemplate } from "../components/Template";
import { useSearchParams } from "@solidjs/router";
import { AppContext } from "../Context";
import { useWebSocket } from "../Websockets";
import FileCard from "../components/FileCard";
import CollectionCard from "../components/CollectionCard";
import Dialog from "@corvu/dialog";
import Dropdown from "../components/Dropdown";
import { createEffect } from "solid-js";
import { getCollection } from "../library/functions";

const Popup: Component<{collectionId: string}> = (props) => {
    const ctx = useContext(AppContext)!;
    const { socket } = useWebSocket();
    const [selectedFolders, setSelectedFolders] = createSignal<string[]>([]);

    const handleAddFolders = (close: () => void) => {
        if (socket()) {
            for (const folderId of selectedFolders()) {
                socket()?.send(JSON.stringify({
                    type: "add_folder_to_collection",
                    data: {
                        collection_id: props.collectionId,
                        folder_id: folderId,
                        auth: {
                            token: localStorage.getItem("token") || "",
                            email: localStorage.getItem("email") || "",
                            password: localStorage.getItem("password") || ""
                        }
                    }
                }));
            }
        }
        setSelectedFolders([]);
        close();
    };

    const availableCollections = () => {
        const currentFolders = ctx.knownCollections()[props.collectionId]?.folders.map(f => f.id) || [];
        return ctx.userCollections().filter(c => c.id !== props.collectionId && !currentFolders.includes(c.id));
    }

    return (
        <Dialog onOpenChange={(open) => !open && setSelectedFolders([])}>
            <Dialog.Trigger class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-green-600 hover:bg-green-800 p-[0.2vh] px-[1vh] rounded-[1vh] font-bold translate-y-[4vh]">
                <span class="text-4xl text-center">+</span>&nbsp;Add Folder
            </Dialog.Trigger>
            <Dialog.Portal>
            <Dialog.Overlay class="fixed inset-0 z-50 bg-black/50 data-open:animate-in data-open:fade-in-0% data-closed:animate-out data-closed:fade-out-0%"/>
            <Dialog.Content class="fixed z-50 top-[50%] left-[50%] translate-x-[-50%] translate-y-[-50%] w-[90vw] max-w-md bg-neutral-800 rounded-lg p-6 space-y-4">
                <p class="text-white text-lg font-bold mb-4 text-center">Add Folder From Collections</p>
                <Dropdown 
                    options={availableCollections()}
                    selected={selectedFolders()}
                    onChange={setSelectedFolders}
                />
                <Dialog.Close>
                    <button 
                        onClick={() => handleAddFolders(() => {})} 
                        class="bg-green-700 text-white p-2 rounded-lg font-semibold w-full hover:bg-green-800 transition-colors disabled:bg-gray-500 disabled:cursor-not-allowed"
                        disabled={selectedFolders().length === 0}
                    >
                        Add
                    </button>
                </Dialog.Close>
            </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    );
}

const CollectionPage: Component = () => {
    const ctx = useContext(AppContext)!;
    const {socket, status} = useWebSocket();

    const [collectionId, setCollectionId] = createSignal<string>("");

    const [params] = useSearchParams();

    createEffect(() => {
        const newCollectionId = params.id?.toString().split(" ").pop() || "";
        if (newCollectionId !== collectionId()) {
            setCollectionId(newCollectionId);
            getCollection(newCollectionId, status, socket, ctx);
        }
    });
    return (
        <DesktopTemplate CurrentPage="Collection">
            <div class="flex flex-col w-full h-full px-[2vh] p-[1vh] space-y-10">
            <div class="w-full flex justify-between items-center">
                <div/>
                <p class="text-white font-black text-[4vh] text-center">{ctx.knownCollections()[collectionId()]?.name || "Unknown Collection"}</p>
                <Popup collectionId={collectionId()} />
            </div>
            <div class="w-full flex flex-wrap justify-center items-start gap-[2vh]">
                <For each={ctx.knownCollections()[collectionId()]?.folders || []}>
                    {(folder) => (
                    <CollectionCard collection={folder} />
                    )}
                </For>
            </div>
            <For each={ctx.knownCollections()[collectionId()]?.files || []}>
                {(file) => (
                <FileCard File={file} />
                )}
            </For>
            </div>
        </DesktopTemplate>
    );
}

export default CollectionPage;