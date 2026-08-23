import { Component, For, createSignal, createEffect, useContext } from "solid-js";
import { useNavigate, useLocation } from "@solidjs/router";
import { AppContext } from "@/Context";
import { useWebSocket } from "@/Websockets";
import { getCollection, getCollectionPathIds } from "@/library/functions";

const CollectionNavigator: Component = () => {
    const [collectionIds, setCollectionIds] = createSignal<Array<string>>([]);
    const location = useLocation();
    const ctx = useContext(AppContext)!;
    const {socket, status} = useWebSocket();
    const navigate = useNavigate();
    createEffect(() => {
        const pathIds = getCollectionPathIds(location.pathname);
        pathIds.pop();
        setCollectionIds(pathIds);
        for (const id of pathIds) {
            if (!ctx.knownCollections()[id]) {
                getCollection(id, status, socket, ctx);
            }
        }
    });

    const handleCollectionClick = (clickedId: string) => {
        const index = collectionIds().indexOf(clickedId);
        if (index !== -1) {
            const newIds = collectionIds().slice(0, index + 1);
            navigate(`/collection/${newIds.join("/")}`);
        }
    };

    return (
        <p class="text-gray-200 hover:cursor-default text-sm md:text-base">
            <span class="font-semibold text-gray-400 hover:text-gray-200 hover:cursor-pointer" onClick={()=>{navigate("/my_collections")}}>Collection</span>
            &nbsp;&nbsp;&#47;&nbsp;&nbsp;
            <For each={collectionIds()}>
                {(id) => (
                    <span
                        class="text-gray-400 hover:text-gray-200 hover:cursor-pointer"
                        onClick={() => handleCollectionClick(id)}
                    >
                        {ctx.knownCollections()[id]?.name || id}
                        &nbsp;&nbsp;&#47;&nbsp;&nbsp;
                    </span>
                )}
            </For>
        </p>
    )
}

export default CollectionNavigator;