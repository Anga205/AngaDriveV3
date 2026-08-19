import { Component, For, useContext, createSignal, createEffect, Show } from "solid-js";
import { useSearchParams } from "@solidjs/router";
import { AppContext } from "@/Context";
import { useWebSocket } from "@/Websockets";
import FileCard from "@/components/FileCard";
import CollectionCard from "@/components/CollectionCard";
import Navbar from "@/components/Navbar";
import { getCollection } from "@/library/functions";
import CollectionNavigator from "../shared/components/CollectionNavigator";
import AddFilePopup from "../shared/components/AddFilePopup";
import AddFolderPopup from "../shared/components/AddFolderPopup";

const CollectionPageMobile: Component = () => {
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

    const hasFolders = () => (ctx.knownCollections()[collectionId()]?.folders?.length || 0) > 0;
    const hasFiles = () => (ctx.knownCollections()[collectionId()]?.files?.length || 0) > 0;

    return (
        <div class="flex flex-col w-full max-h-screen h-screen bg-black">
            <Navbar CurrentPage="Collections" Type="mobile"/>
            <div class="h-[6vh]"/>
            <div class="px-3 space-y-2">
                <CollectionNavigator />
                <p class="text-white font-black text-[4vh]">{ctx.knownCollections()[collectionId()]?.name || "Unknown Collection"}</p>
                <Show when={ctx.knownCollections()[collectionId()]?.isOwned}>
                    <div class="flex justify-end space-x-2">
                        <AddFolderPopup collectionId={collectionId()} isMobile={true} />
                        <AddFilePopup collectionId={collectionId()} isMobile={true} />
                    </div>
                </Show>
            </div>
            <div class="w-full px-4 mt-4 max-h-full h-full flex flex-col space-y-4 overflow-y-auto custom-scrollbar">
                <Show when={hasFolders() && hasFiles()}>
                    <p class="text-gray-400 text-lg font-semibold w-full text-center">Folders</p>
                </Show>
                <Show when={hasFolders()}>
                    <div class="w-full flex flex-wrap justify-center gap-4">
                        <For each={ctx.knownCollections()[collectionId()]?.folders || []}>
                            {(folder) => <CollectionCard collection={folder} />}
                        </For>
                    </div>
                </Show>
                <Show when={hasFolders() && hasFiles()}>
                     <p class="text-gray-400 text-lg font-semibold w-full text-center">Files</p>
                </Show>
                <Show when={hasFiles()}>
                     <div class="w-full flex flex-wrap justify-center gap-4 pt-6">
                        <For each={ctx.knownCollections()[collectionId()]?.files || []}>
                            {(file) => <FileCard File={file} />}
                        </For>
                    </div>
                </Show>
                <div class="w-full h-[2vh]"/>
            </div>
        </div>
    );
}

export default CollectionPageMobile;