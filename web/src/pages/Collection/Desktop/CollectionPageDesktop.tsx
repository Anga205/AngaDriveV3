import { Component, For, useContext, createSignal, createEffect, Show } from "solid-js";
import { DesktopTemplate } from "@/components/Template";
import { useSearchParams } from "@solidjs/router";
import { AppContext } from "@/Context";
import { useWebSocket } from "@/Websockets";
import FileCard from "@/components/FileCard";
import CollectionCard from "@/components/CollectionCard";
import { getCollection } from "@/library/functions";
import CollectionNavigator from "../shared/components/CollectionNavigator";
import AddFilePopup from "../shared/components/AddFilePopup";
import AddFolderPopup from "../shared/components/AddFolderPopup";

const CollectionPageDesktop: Component = () => {
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
                    <CollectionNavigator />
                    <p class="text-white font-black text-[4vh] text-center">{ctx.knownCollections()[collectionId()]?.name || "Unknown Collection"}</p>
                    {
                        ctx.knownCollections()[collectionId()]?.isOwned ?
                        <div class="flex space-x-2">
                            <AddFolderPopup collectionId={collectionId()} />
                            <AddFilePopup collectionId={collectionId()} />
                        </div>
                        :
                        <div/>
                    }
                </div>
                <div class="w-full flex flex-col overflow-y-scroll space-y-10 custom-scrollbar h-full">
                    <Show when={(ctx.knownCollections()[collectionId()]?.folders?.length || 0) > 0 && (ctx.knownCollections()[collectionId()]?.files?.length || 0) > 0}>
                        <div class="w-full flex justify-center items-center">
                            <hr class="border-t border-gray-600 w-full" />
                            <p class="mx-4 text-gray-400">Folders</p>
                            <hr class="border-t border-gray-600 w-full" />
                        </div>
                    </Show>
                    <div class="w-full flex flex-wrap justify-center items-start gap-[2vh]">
                        <For each={ctx.knownCollections()[collectionId()]?.folders || []}>
                            {(folder) => (
                            <CollectionCard collection={folder} />
                            )}
                        </For>
                    </div>
                    <Show when={(ctx.knownCollections()[collectionId()]?.folders?.length || 0) > 0 && (ctx.knownCollections()[collectionId()]?.files?.length || 0) > 0}>
                        <div class="w-full flex justify-center items-center">
                            <hr class="border-t border-gray-600 w-full" />
                            <p class="mx-4 text-gray-400">Files</p>
                            <hr class="border-t border-gray-600 w-full" />
                        </div>
                    </Show>
                    <div class="w-full flex flex-wrap justify-center items-start gap-[2vh]">
                        <For each={ctx.knownCollections()[collectionId()]?.files || []}>
                            {(file) => (
                            <FileCard File={file} />
                            )}
                        </For>
                    </div>
                </div>
            </div>
        </DesktopTemplate>
    );
}

export default CollectionPageDesktop;