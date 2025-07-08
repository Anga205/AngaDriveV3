import { Component, For, onMount, useContext } from "solid-js";
import { DesktopTemplate } from "../components/Template";
import { useSearchParams } from "@solidjs/router";
import { AppContext } from "../Context";
import { useWebSocket } from "../Websockets";
import FileCard from "../components/FileCard";
import CollectionCard from "../components/CollectionCard";

const CollectionPage: Component = () => {
    const [params] = useSearchParams();
    const ctx = useContext(AppContext)!;
    const {socket, status} = useWebSocket();
    const collectionId = params.id?.toString() || "";
    const getCollection = () => {
        if (status() !== "connected") {
            setTimeout(getCollection, 1000);
        } else {
            socket()?.send(JSON.stringify({
                type: "get_collection",
                data: {
                    id: collectionId,
                    auth: {
                        token: localStorage.getItem("token") || "",
                        email: localStorage.getItem("email") || "",
                        password: localStorage.getItem("password") || ""
                    }
                }
            }));
        }
    }
    onMount(() => {
        getCollection();
    })
    return (
        <DesktopTemplate CurrentPage="Collection">
            <div class="flex flex-col w-full h-full px-[2vh] p-[1vh] space-y-10">
                <div class="w-full flex justify-between items-center">
                    <p class="text-white font-black text-[4vh] text-center w-full">{ctx.knownCollections()[collectionId]?.name || "Unknown Collection"}</p>
                </div>
                <For each={ctx.knownCollections()[collectionId]?.folders || []}>
                    {(folder) => (
                        <CollectionCard collection={folder} />
                    )}
                </For>
                <For each={ctx.knownCollections()[collectionId]?.files || []}>
                    {(file) => (
                        <FileCard File={file} />
                    )}
                </For>
            </div>
        </DesktopTemplate>
    );
}

export default CollectionPage;