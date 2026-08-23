import { For, Show, Accessor, Component, createEffect, createMemo, createSignal, onCleanup, onMount, useContext } from "solid-js";
import { FileData } from "@/library/types";
import Select, { SelectOption } from "@/components/Select";
import { AppContext } from "@/Context";
import Navbar from "@/components/Navbar";
import Search from "lucide-solid/icons/search";
import { UploadPopup } from "../shared/components/UploadPopUp";
import FilesError from "../shared/components/FilesError";
import FileCard from "@/components/FileCard";
import { useWebSocket } from "@/Websockets";
import { toast } from "solid-toast";
import BulkDeleteDialog from "../shared/components/BulkDeleteDialog";
import { CrossSVG } from "@/assets/SvgFiles";

const MobileDrive: Component<{Files: Accessor<Array<FileData>>; sortOptions: SelectOption[]; selectedSort: Accessor<string[]>; setSelectedSort: (value: string[]) => void; sortedFiles: () => Array<FileData>; searchQuery?: Accessor<string>; setSearch?: (v: string) => void}> = (props) => {
    // Lazy load for mobile as well
    const [visibleCount, setVisibleCount] = createSignal(50);
    let mobileScrollRef: HTMLDivElement | undefined;
    let mobileSentinelRef: HTMLDivElement | undefined;
    let mobileObserver: IntersectionObserver | undefined;

    const ctx = useContext(AppContext)!;
    const { socket: getSocket } = useWebSocket();
    const [deleteOpen, setDeleteOpen] = createSignal(false);

    const selectedCount = createMemo(() => ctx.selectedFiles?.()?.size || 0);

    const handleBulkDelete = () => {
        const selected = Array.from(ctx.selectedFiles?.() || new Set<string>());
        if (selected.length === 0) return;
        const deleteRequest = {
            type: "bulk_delete_files",
            data: {
                file_directories: selected,
                auth: {
                    token: localStorage.getItem("token") || "",
                    email: localStorage.getItem("email") || "",
                    password: localStorage.getItem("password") || ""
                }
            }
        }
        if (getSocket()?.readyState !== WebSocket.OPEN) {
            toast.error("WebSocket is not available");
            return;
        }
        getSocket()?.send(JSON.stringify(deleteRequest));
        ctx.setSelectedFiles?.(new Set());
        setDeleteOpen(false);
    };

    const handleUnselectAll = () => {
        ctx.setSelectedFiles?.(new Set());
    };

    const displayedFiles = createMemo(() => {
        const base = props.sortedFiles() || [];
        const visible = base.slice(0, visibleCount());
        const loaded = Array.from(ctx.loadedFiles?.() || new Set<string>());
        const extras = base.filter(f => loaded.includes(f.file_directory) && !visible.some(v => v.file_directory === f.file_directory));
        return [...visible, ...extras];
    });

    // Reset the number of visible files only when the sort order changes.
    // (Not on every file list update, otherwise re-renders caused by uploads
    // or deletes would make already-loaded files disappear.)
    createEffect(() => {
        props.selectedSort();
        setVisibleCount(50);
    });

    onMount(() => {
        mobileObserver = new IntersectionObserver(
            (entries) => {
                for (const entry of entries) {
                    if (entry.isIntersecting) {
                        const total = props.sortedFiles().length;
                        if (visibleCount() < total) {
                            setVisibleCount(Math.min(visibleCount() + 20, total));
                        }
                    }
                }
            },
            { root: mobileScrollRef, rootMargin: "0px 0px 200px 0px", threshold: 0 }
        );
        if (mobileSentinelRef) mobileObserver.observe(mobileSentinelRef);
    });

    onCleanup(() => mobileObserver?.disconnect());

    return (
        <div class="flex flex-col w-full max-h-screen h-screen bg-black">
            <Navbar CurrentPage="Files" Type="mobile"/>
            <div class="h-[6vh]"/>
            <p class="text-white font-black text-[4vh] px-3">My&nbsp;Files</p>
            <div class="flex flex-col justify-between items-center gap-3 px-3">
                <Show when={props.Files().length >= 2}>
                    <div class="flex items-center w-full">
                        <div class="relative w-full">
                            <input
                                class="w-full bg-neutral-900 placeholder-neutral-500 text-neutral-200 rounded-lg px-3 py-2 pr-10 border border-neutral-800 focus:outline-none"
                                placeholder="Search files by name or path"
                                value={props.searchQuery ? props.searchQuery() : ''}
                                onInput={(e) => props.setSearch ? props.setSearch((e.target as HTMLInputElement).value) : null}
                            />
                            <div class="absolute right-2 top-2 text-neutral-400">
                                <Search class="w-5 h-5" />
                            </div>
                        </div>
                    </div>
                </Show>
                <div class={`flex gap-3 w-full ${props.Files().length >= 2 ? 'justify-between' : 'justify-end'}`}>
                    <Show when={props.Files().length >= 2}>
                        <div class="z-5 h-full">
                            <Select
                                options={props.sortOptions}
                                selected={props.selectedSort()}
                                onChange={(s) => props.setSelectedSort(s.length ? [s[s.length - 1]] : [])}
                                placeholderText="Sort By"
                            />
                        </div>
                    </Show>
                    <UploadPopup/>
                </div>
            <Show when={selectedCount() > 0}>
                <div class="w-full flex items-center gap-2 px-3 py-2 bg-neutral-900 border border-neutral-800 rounded-lg">
                    <p class="text-neutral-300 text-sm font-semibold">{selectedCount()} selected</p>
                    <div class="flex-1" />
                    <button
                        class="flex items-center gap-1 px-2 py-1 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-medium transition-colors duration-150"
                        onClick={handleUnselectAll}
                    >
                        <CrossSVG />
                        Unselect
                    </button>
                    <BulkDeleteDialog
                        open={deleteOpen()}
                        onOpenChange={setDeleteOpen}
                        onDelete={handleBulkDelete}
                        selectedCount={selectedCount()}
                        triggerLabel="Delete"
                    />
                </div>
            </Show>
            </div>
            <div ref={(el) => (mobileScrollRef = el)} class="w-full px-4 mt-4 max-h-full h-full flex flex-wrap items-center space-y-4 space-x-4 justify-center overflow-y-auto">
                <For each={displayedFiles()} fallback={<FilesError />}>
                    {(file) => (
                        <FileCard
                            File={file}
                            onSelectionToggle={(directory) => {
                                ctx.setSelectedFiles?.((prev) => {
                                    const next = new Set(prev);
                                    if (next.has(directory)) next.delete(directory);
                                    else next.add(directory);
                                    return next;
                                });
                            }}
                            isSelected={ctx.selectedFiles?.()?.has(file.file_directory) || false}
                        />
                    )}
                </For>
                {/* Sentinel for lazy loading */}
                <div ref={(el) => (mobileSentinelRef = el)} class="w-full h-px" />
                <div class="w-full h-[2vh]"/>
            </div>
        </div>
    )
}

export default MobileDrive;