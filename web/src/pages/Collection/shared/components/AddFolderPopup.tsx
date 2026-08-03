import { Component, useContext, createSignal, createEffect } from "solid-js";
import Dialog from "@corvu/dialog";
import Dropdown from "@/components/Dropdown";
import { AppContext } from "@/Context";
import { useWebSocket } from "@/Websockets";
import { CollectionCardData } from "@/library/types";

const AddFolderPopup: Component<{collectionId: string, isMobile?: boolean}> = (props) => {
    const ctx = useContext(AppContext)!;
    const { socket } = useWebSocket();
    const [selectedFolders, setSelectedFolders] = createSignal<string[]>([]);
    const [newFolderName, setNewFolderName] = createSignal("");
    const [modifying, setModifying] = createSignal<"existing" | "new" | null>(null);

    createEffect(() => {
        if (selectedFolders().length > 0) {
            setModifying("existing");
        } else if (newFolderName().trim() !== "") {
            setModifying("new");
        } else {
            setModifying(null);
        }
    });

    const handleSubmit = (close: () => void) => {
        if (socket()) {
            if (modifying() === "existing") {
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
            } else if (modifying() === "new" && newFolderName().trim() !== "") {
                socket()?.send(JSON.stringify({
                    type: "create_folder_in_collection",
                    data: {
                        collection_id: props.collectionId,
                        folder_name: newFolderName().trim(),
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
        setNewFolderName("");
        close();
    };

    const availableCollections = () => {
        const currentFolders = ctx.knownCollections()[props.collectionId]?.folders.map(f => f.id) || [];
        const collections: CollectionCardData[] = []
        for (const collection of ctx.userCollections()) {
            if (ctx.knownCollectionCards()[collection].id !== props.collectionId && !currentFolders.includes(ctx.knownCollectionCards()[collection].id)) {
                collections.push(ctx.knownCollectionCards()[collection]);
            }
        }
        return collections;
    }

    const isButtonVisible = () => {
        return modifying() !== null;
    }

    return (
        <Dialog onOpenChange={(open) => {
            if (!open) {
                setSelectedFolders([]);
                setNewFolderName("");
            }
        }}>
            <Dialog.Trigger class={`cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-green-600 hover:bg-green-800 p-[0.2vh] px-[1vh] rounded-[1vh] font-bold ${!props.isMobile && 'translate-y-[4vh]'}`}>
                <span class="text-4xl text-center">+</span>&nbsp;Add Folder
            </Dialog.Trigger>
            <Dialog.Portal>
            <Dialog.Overlay class="fixed inset-0 z-50 bg-black/50 data-open:animate-in data-open:fade-in-0% data-closed:animate-out data-closed:fade-out-0%"/>
            <Dialog.Content class="fixed z-50 top-[50%] left-[50%] translate-x-[-50%] translate-y-[-50%] w-[90vw] max-w-md bg-neutral-800 rounded-lg p-6 space-y-4">

                {(modifying() === "new" || modifying() === null) && (
                    <>
                        <p class="text-white text-lg font-bold mb-2 text-center">Create New Folder</p>
                        <input
                            type="text"
                            placeholder="New folder name..."
                            value={newFolderName()}
                            onInput={(e) => setNewFolderName(e.currentTarget.value)}
                            class="w-full bg-neutral-700 text-white p-2 rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500"
                        />
                    </>
                )}
                {modifying() === null && (
                    <div class="flex w-full items-center justify-center">
                        <hr class="w-full border-neutral-600"/>
                        <p class="mx-2 text-gray-500">OR</p>
                        <hr class="w-full border-neutral-600"/>
                    </div>
                )}

                {(modifying() === "existing" || modifying() === null) && (
                    <>
                        <p class="text-white text-lg font-bold mb-2 text-center">Add Existing Folder</p>
                        <Dropdown
                            options={availableCollections()}
                            selected={selectedFolders()}
                            onChange={setSelectedFolders}
                            placeholderText="Select Folders"
                        />
                    </>
                )}

                {isButtonVisible() && (
                    <Dialog.Close class="w-full">
                        <button
                            onClick={() => handleSubmit(() => {})}
                            class="bg-green-700 text-white p-2 rounded-lg font-semibold w-full hover:bg-green-800 transition-colors disabled:bg-gray-500 disabled:cursor-not-allowed"
                            disabled={modifying() === "existing" ? selectedFolders().length === 0 : newFolderName().trim().length <= 2}
                        >
                            {modifying() === "new" ? "Create" : "Add Selected"}
                        </button>
                    </Dialog.Close>
                )}
            </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    );
}

export default AddFolderPopup;