import { Component, For, useContext, createSignal, createEffect, Show, onMount, onCleanup } from "solid-js";
import { DesktopTemplate } from "../components/Template";
import { useNavigate, useSearchParams } from "@solidjs/router";
import { AppContext } from "../Context";
import { useWebSocket } from "../Websockets";
import FileCard from "../components/FileCard";
import CollectionCard from "../components/CollectionCard";
import Dialog from "@corvu/dialog";
import Dropdown from "../components/Dropdown";
import { getCollection, generateUUID } from "../library/functions";
import { Toaster } from "solid-toast";
import { CollectionCardData } from "../library/types";
import { uploadFileInChunks, FileUploadPreview, SelectableFile, FileUploadProgressData } from "./MyDrive";
import Navbar from "../components/Navbar";

const AddFilePopup: Component<{collectionId: string, isMobile?: boolean}> = (props) => {
    const ctx = useContext(AppContext)!;
    const { socket } = useWebSocket();
    const [selectedExistingFiles, setSelectedExistingFiles] = createSignal<string[]>([]);
    const [selectedUploadFiles, setSelectedUploadFiles] = createSignal<SelectableFile[]>([]);
    const [uploadProgressMap, setUploadProgressMap] = createSignal<Record<string, FileUploadProgressData>>({});
    const [isUploading, setIsUploading] = createSignal(false);
    const [modifying, setModifying] = createSignal<"existing" | "new" | null>(null);
    const [isDragOver, setIsDragOver] = createSignal(false);
    const [open, setOpen] = createSignal(false);

    createEffect(() => {
        if (selectedExistingFiles().length > 0) {
            setModifying("existing");
        } else if (selectedUploadFiles().length > 0) {
            setModifying("new");
        } else {
            setModifying(null);
        }
    });

    const getPathKey = (f: File) => ((f as any).webkitRelativePath || (f as any).path || f.name);

    const addFiles = (files: FileList | File[]) => {
        const list = Array.from(files);
        if (list.length === 0) return;
        const existingKeys = new Set(selectedUploadFiles().map(sf => getPathKey(sf.file)));
        const deduped = list
            .filter(f => !existingKeys.has(getPathKey(f)))
            .map(f => ({ uniqueId: generateUUID(), file: f }));
        if (deduped.length === 0) return;
        setSelectedUploadFiles(prev => [...prev, ...deduped]);
        setUploadProgressMap(prevMap => {
            const updatedMap = { ...prevMap };
            deduped.forEach(sf => {
                if (!updatedMap[sf.uniqueId]) {
                    updatedMap[sf.uniqueId] = {
                        id: sf.uniqueId,
                        name: sf.file.name,
                        progress: 0,
                        status: 'pending',
                    };
                }
            });
            return updatedMap;
        });
    };

    const handleFileChange = (event: Event) => {
        const input = event.target as HTMLInputElement;
        if (input.files) {
            addFiles(input.files);
        }
        if (input) input.value = '';
    };

    const addDroppedFiles = (files: FileList | File[]) => addFiles(files);

    onMount(() => {
        const handler = (e: Event) => {
            const ce = e as CustomEvent<{ files?: File[] | FileList }>;
            const files = ce.detail?.files;
            if (files && (files as any).length !== undefined) {
                addDroppedFiles(files as File[] | FileList);
            }
            setOpen(true);
            setModifying("new");
        };
        document.addEventListener("open-collection-upload", handler as EventListener);
        queueMicrotask(() => {
            if (!open()) {
                const pending = (window as any).__pendingCollectionUploadFiles as File[] | undefined;
                if (pending && pending.length) {
                    addDroppedFiles(pending);
                    (window as any).__pendingCollectionUploadFiles = undefined;
                    setOpen(true);
                    setModifying("new");
                }
            }
        });
        onCleanup(() => document.removeEventListener("open-collection-upload", handler as EventListener));
    });

    const handleFileDelete = (uniqueIdToDelete: string) => {
        setSelectedUploadFiles((prev) => prev.filter(sf => sf.uniqueId !== uniqueIdToDelete));
        setUploadProgressMap(prev => {
            const updated = { ...prev };
            delete updated[uniqueIdToDelete];
            return updated;
        });
    };

    const handleUpload = async () => {
        let authDetails: any = {};
        const storedEmail = localStorage.getItem("email");
        const storedPassword = localStorage.getItem("password");

        if (storedEmail && storedPassword) {
            authDetails = { email: storedEmail, password: storedPassword };
        } else {
            authDetails = { token: localStorage.getItem("token") || "" };
        }

        setIsUploading(true);
        const filesToProcess = selectedUploadFiles().filter(sf => {
            const currentStatus = uploadProgressMap()[sf.uniqueId]?.status;
            return currentStatus === 'pending' || currentStatus === 'error';
        });

        const queue = [...filesToProcess];
        let activeUploads = 0;
        const MAX_CONCURRENT_UPLOADS = 3;

        const processNext = async () => {
            if (activeUploads >= MAX_CONCURRENT_UPLOADS || queue.length === 0) {
                if (activeUploads === 0 && queue.length === 0) setIsUploading(false);
                return;
            }

            activeUploads++;
            const selectableFile = queue.shift();
            if (!selectableFile) {
                activeUploads--;
                processNext();
                return;
            }

            setUploadProgressMap(prev => ({ ...prev, [selectableFile.uniqueId]: { ...prev[selectableFile.uniqueId], status: 'uploading', progress: 0 } }));
            
            try {
                await uploadFileInChunks(selectableFile, generateUUID(), authDetails, (progress) => {
                    setUploadProgressMap(prev => ({ ...prev, [selectableFile.uniqueId]: { ...prev[selectableFile.uniqueId], progress } }));
                }, props.collectionId);
                setUploadProgressMap(prev => ({ ...prev, [selectableFile.uniqueId]: { ...prev[selectableFile.uniqueId], status: 'completed', progress: 100 } }));
            } catch (error: any) {
                setUploadProgressMap(prev => ({ ...prev, [selectableFile.uniqueId]: { ...prev[selectableFile.uniqueId], status: 'error', errorMessage: error.message || 'Upload failed' } }));
            } finally {
                activeUploads--;
                processNext();
            }
        };

        for (let i = 0; i < MAX_CONCURRENT_UPLOADS; i++) {
            processNext();
        }
    };

    const handleSubmit = (close: () => void) => {
        if (modifying() === "existing" && socket()) {
            for (const fileDirectory of selectedExistingFiles()) {
                socket()?.send(JSON.stringify({
                    type: "add_file_to_collection",
                    data: {
                        collection_id: props.collectionId,
                        file_directory: fileDirectory,
                        auth: {
                            token: localStorage.getItem("token") || "",
                            email: localStorage.getItem("email") || "",
                            password: localStorage.getItem("password") || ""
                        }
                    }
                }));
            }
        } else if (modifying() === "new") {
            handleUpload();
        }
        setSelectedExistingFiles([]);
        // Don't close automatically for uploads, let user see progress.
        if (modifying() === "existing") {
            close();
        }
    };

    const availableFiles = () => {
        const currentFiles = ctx.knownCollections()[props.collectionId]?.files.map(f => f.file_directory) || [];
        return ctx.files().filter(f => !currentFiles.includes(f.file_directory)).map(f => ({id: f.file_directory, name: f.original_file_name}));
    }

    const filesPendingOrError = () => selectedUploadFiles().filter(sf => {
        const status = uploadProgressMap()[sf.uniqueId]?.status;
        return status === 'pending' || status === 'error';
    }).length;

    return (
        <Dialog open={open()} onOpenChange={(o) => {
            setOpen(o);
            if (!o) {
                setSelectedExistingFiles([]);
                setSelectedUploadFiles([]);
                setUploadProgressMap({});
                setIsUploading(false);
            }
        }}>
            <Dialog.Trigger class={`cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-blue-600 hover:bg-blue-800 p-[0.2vh] px-[1vh] rounded-[1vh] font-bold ${!props.isMobile && 'translate-y-[4vh]'}`}>
                <span class="text-4xl text-center">+</span>&nbsp;Add File
            </Dialog.Trigger>
            <Dialog.Portal>
            <Dialog.Overlay class="fixed inset-0 z-50 bg-black/50 data-open:animate-in data-open:fade-in-0% data-closed:animate-out data-closed:fade-out-0%"/>
            <Dialog.Content class="fixed z-50 top-[50%] left-[50%] translate-x-[-50%] translate-y-[-50%] w-[90vw] max-w-lg bg-neutral-800 rounded-lg p-6 space-y-4">
                {(modifying() === "new" || modifying() === null) && (
                    <>
                        <p class="text-white text-lg font-bold mb-2 text-center">Upload New File</p>
                        <label
                            for="collection-file-upload"
                            class={`rounded-md min-h-[15vh] flex justify-center items-center cursor-pointer my-[1vh] ${selectedUploadFiles().length === 0 ? `border-2 ${isDragOver() ? 'border-blue-400' : 'border-dotted border-blue-800'}` : ''}`}
                            onDragOver={(e) => { e.preventDefault(); setIsDragOver(true); }}
                            onDragEnter={(e) => { e.preventDefault(); setIsDragOver(true); }}
                            onDragLeave={() => setIsDragOver(false)}
                            onDrop={(e) => {
                                e.preventDefault();
                                setIsDragOver(false);
                                if (e.dataTransfer?.files && e.dataTransfer.files.length > 0) {
                                    addDroppedFiles(e.dataTransfer.files);
                                }
                            }}
                        >
                            {selectedUploadFiles().length === 0 ? (
                                <p class="text-center p-4 text-white">Drag and drop files here or click to select files</p>
                            ) : (
                                <div class="flex flex-col w-full space-y-2 max-h-[30vh] overflow-y-auto custom-scrollbar p-1">
                                    <For each={selectedUploadFiles()}>
                                        {(sf) => (
                                            <FileUploadPreview
                                                selectableFile={sf}
                                                uploadInfo={() => uploadProgressMap()[sf.uniqueId]}
                                                onDelete={handleFileDelete}
                                            />
                                        )}
                                    </For>
                                </div>
                            )}
                            <input id="collection-file-upload" type="file" multiple class="hidden" onChange={handleFileChange} />
                        </label>
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
                        <p class="text-white text-lg font-bold mb-2 text-center">Add Existing File</p>
                        <Dropdown 
                            options={availableFiles()}
                            selected={selectedExistingFiles()}
                            onChange={setSelectedExistingFiles}
                            placeholderText="Select Files"
                        />
                    </>
                )}

                <Show when={
                    (selectedUploadFiles().length > 0 && filesPendingOrError() > 0) || 
                    (modifying() === "existing" && selectedExistingFiles().length > 0)
                }>
                    {modifying() !== null && (
                        <button
                            onClick={() => handleSubmit(() => {})}
                            class="bg-green-700 text-white p-2 rounded-lg font-semibold w-full hover:bg-green-800 transition-colors disabled:bg-gray-500 disabled:cursor-not-allowed mt-4"
                            disabled={
                                (modifying() === 'existing' && selectedExistingFiles().length === 0) ||
                                (modifying() === 'new' && (isUploading() || filesPendingOrError() === 0))
                            }
                        >
                            {modifying() === 'new' ? (isUploading() ? 'Uploading...' : `Upload ${filesPendingOrError()} File(s)`) : 'Add Selected'}
                        </button>
                    )}
                </Show>
                 {modifying() === 'new' && selectedUploadFiles().length > 0 && filesPendingOrError() === 0 && !isUploading() && (
                    <p class="mt-4 text-center text-green-500">All selected files uploaded!</p>
                )}
            </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    );
}

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

const CollectionNavigator: Component = () => {
    const [collectionIds, setCollectionIds] = createSignal<Array<string>>([]);
    const [params] = useSearchParams();
    const ctx = useContext(AppContext)!;
    const {socket, status} = useWebSocket();
    const navigate = useNavigate();
    createEffect(() => {
        const allIds = params.id?.toString().split(" ") || []
        allIds.pop();
        setCollectionIds(allIds);
        for (const id of collectionIds()) {
            if (!ctx.knownCollections()[id]) {
                getCollection(id, status, socket, ctx);
            }
        }
    });

    const handleCollectionClick = (clickedId: string) => {
        const index = collectionIds().indexOf(clickedId);
        if (index !== -1) {
            const newIds = collectionIds().slice(0, index + 1);
            navigate(`/collection?id=${newIds.join(" ")}`);
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

const CollectionPage: Component = () => {
    const [isMobile, setIsMobile] = createSignal(window.innerWidth <= 768);

    const handleResize = () => {
        setIsMobile(window.innerWidth <= 768);
    };

    onMount(() => {
        window.addEventListener('resize', handleResize);
    });

    onCleanup(() => {
        window.removeEventListener('resize', handleResize);
    });

    return (
        <>
            <title>Collection | DriveV3</title>
            {isMobile() ? <CollectionPageMobile /> : <CollectionPageDesktop />}
            <Toaster
            position="bottom-right"
            gutter={8}
            containerClassName=""
            containerStyle={{}}
            toastOptions={{
                className: '',
                duration: 2000,
                style: {
                background: '#363636',
                color: '#fff',
                },
            }}
            />
        </>
    );
}

export default CollectionPage;