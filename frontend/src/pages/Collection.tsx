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
    const [isPaused, setIsPaused] = createSignal(false);
    const [modifying, setModifying] = createSignal<"existing" | "new" | null>(null);
    const [isDragOver, setIsDragOver] = createSignal(false);
    const [open, setOpen] = createSignal(false);
    const activeControllers = new Set<AbortController>();
    const cancelledFiles = new Set<string>();
    const manageController = (c: AbortController, action: 'add' | 'remove') => {
        if (action === 'add') activeControllers.add(c);
        else activeControllers.delete(c);
    };

    // Concurrency-aware scheduler state
    const [activeUploadsCount, setActiveUploadsCount] = createSignal(0);
    const MAX_CONCURRENT_UPLOADS = 3;

    // Debounced queue pumping to avoid microtask storms when many files are pending
    let pumpScheduled = false;
    const requestPump = () => {
        if (pumpScheduled) return;
        pumpScheduled = true;
        queueMicrotask(() => {
            pumpScheduled = false;
            pumpQueue();
        });
    };

    const waitWhilePaused = async () => {
        while (isPaused()) {
            await new Promise(r => setTimeout(r, 200));
        }
    };

    const buildAuthDetails = () => {
        const storedEmail = localStorage.getItem("email");
        const storedPassword = localStorage.getItem("password");
        if (storedEmail && storedPassword) {
            return { email: storedEmail, password: storedPassword };
        }
        return { token: localStorage.getItem("token") || "" };
    };

    const startSingleUpload = async (sf: SelectableFile) => {
        if (cancelledFiles.has(sf.uniqueId)) return;

        const auth = buildAuthDetails();
        // Mark as uploading (preserve progress when retrying)
        setUploadProgressMap(prev => {
            if (cancelledFiles.has(sf.uniqueId)) return prev;
            return ({
                ...prev,
                [sf.uniqueId]: {
                    ...prev[sf.uniqueId],
                    status: 'uploading',
                    progress: prev[sf.uniqueId]?.progress ?? 0,
                }
            });
        });

        setActiveUploadsCount(c => c + 1);
        try {
            await uploadFileInChunks(
                sf,
                generateUUID(),
                auth,
                (progress) => {
                    setUploadProgressMap(prev => {
                        if (cancelledFiles.has(sf.uniqueId)) return prev;
                        return ({ ...prev, [sf.uniqueId]: { ...prev[sf.uniqueId], progress } });
                    });
                },
                props.collectionId,
                waitWhilePaused,
                () => isPaused(),
                manageController,
                () => cancelledFiles.has(sf.uniqueId)
            );
            setUploadProgressMap(prev => {
                if (cancelledFiles.has(sf.uniqueId)) return prev;
                return ({ ...prev, [sf.uniqueId]: { ...prev[sf.uniqueId], status: 'completed', progress: 100 } });
            });
        } catch (error: any) {
            setUploadProgressMap(prev => {
                if (cancelledFiles.has(sf.uniqueId)) return prev;
                return ({ ...prev, [sf.uniqueId]: { ...prev[sf.uniqueId], status: 'error', errorMessage: error?.message || 'Upload failed' } });
            });
        } finally {
            setActiveUploadsCount(c => Math.max(0, c - 1));
            requestPump();
        }
    };

    const pumpQueue = () => {
        if (!open()) return;
        if (isPaused()) return;

        const currentMap = uploadProgressMap();
        const available = Math.max(0, MAX_CONCURRENT_UPLOADS - activeUploadsCount());
        if (available === 0) {
            setIsUploading(true);
            return;
        }

        const candidates = selectedUploadFiles().filter(sf => {
            const st = currentMap[sf.uniqueId]?.status ?? 'pending';
            return (st === 'pending' || st === 'error') && !cancelledFiles.has(sf.uniqueId);
        }).slice(0, available);

        if (candidates.length === 0) {
            if (activeUploadsCount() === 0) setIsUploading(false);
            return;
        }

        setIsUploading(true);
        candidates.forEach(sf => {
            // reset to pending and clear error before start
            setUploadProgressMap(prev => ({
                ...prev,
                [sf.uniqueId]: {
                    ...prev[sf.uniqueId],
                    status: 'pending',
                    progress: prev[sf.uniqueId]?.status === 'error' ? 0 : (prev[sf.uniqueId]?.progress ?? 0),
                    errorMessage: undefined,
                }
            }));
            startSingleUpload(sf);
        });
    };

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
        if (!isPaused()) requestPump();
    };

    const handleFileChange = (event: Event) => {
        const input = event.target as HTMLInputElement;
        if (input.files) {
            addFiles(input.files);
        }
        if (input) input.value = '';
    };

    const addDroppedFiles = (files: FileList | File[]) => { addFiles(files); if (open() && !isPaused()) requestPump(); };

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
        cancelledFiles.add(uniqueIdToDelete);
        setSelectedUploadFiles((prev) => prev.filter(sf => sf.uniqueId !== uniqueIdToDelete));
        setUploadProgressMap(prev => {
            const updated = { ...prev };
            delete updated[uniqueIdToDelete];
            return updated;
        });
    };

    // Removed old one-shot handler in favor of pumpQueue/startSingleUpload

    // Auto-start uploads when there are pending items and not paused
    createEffect(() => {
        if (open() && modifying() === 'new' && selectedUploadFiles().length > 0) {
            const pendingCount = selectedUploadFiles().filter(sf => {
                const status = uploadProgressMap()[sf.uniqueId]?.status;
                return status === 'pending' || status === 'error';
            }).length;
            if (pendingCount > 0 && !isPaused()) {
                requestPump();
            }
        }
    });

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
            requestPump();
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

    const anyUploading = () => selectedUploadFiles().some(sf => uploadProgressMap()[sf.uniqueId]?.status === 'uploading');

    return (
        <Dialog open={open()} onOpenChange={(o) => {
            setOpen(o);
            if (!o) {
                setSelectedExistingFiles([]);
                setSelectedUploadFiles([]);
                setUploadProgressMap({});
                setIsUploading(false);
                setIsPaused(false);
                activeControllers.forEach(c => c.abort());
                activeControllers.clear();
                cancelledFiles.clear();
                pumpScheduled = false;
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
                                                canDelete={() => isPaused()}
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

                <Show when={(modifying() === 'existing' && selectedExistingFiles().length > 0)}>
                    <button
                        onClick={() => handleSubmit(() => {})}
                        class="bg-green-700 text-white p-2 rounded-lg font-semibold w-full hover:bg-green-800 transition-colors disabled:bg-gray-500 disabled:cursor-not-allowed mt-4"
                        disabled={selectedExistingFiles().length === 0}
                    >
                        Add Selected
                    </button>
                </Show>

                <Show when={modifying() === 'new' && selectedUploadFiles().length > 0 && (filesPendingOrError() > 0 || anyUploading())}>
                    <button
                        class={`${isPaused() ? 'bg-blue-600 hover:bg-blue-800' : 'bg-yellow-600 hover:bg-yellow-800'} disabled:bg-gray-600 text-white font-bold py-2 px-4 rounded w-full mt-4`}
                        onClick={() => {
                            const next = !isPaused();
                            setIsPaused(next);
                            if (next) {
                                // Pause: abort in-flight
                                activeControllers.forEach(c => c.abort());
                            } else {
                                // Resume: pump queue immediately
                                requestPump();
                            }
                        }}
                    >
                        {isPaused() ? 'Resume' : 'Pause'}
                    </button>
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