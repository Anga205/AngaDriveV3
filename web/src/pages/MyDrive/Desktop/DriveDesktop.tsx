import type { Accessor, Component } from "solid-js"
import { DesktopTemplate } from "@/components/Template"
import { createSignal, Show, For, createMemo, onCleanup, createEffect, useContext, onMount } from "solid-js"
import Select, { SelectOption } from "@/components/Select";
import { AppContext } from "@/Context";
import { FileData } from "@/library/types";
import Search from "lucide-solid/icons/search";
import { UploadPopup } from "../shared/components/UploadPopUp";
import FilesError from "../shared/components/FilesError";
import FileCard from "@/components/FileCard";

const DesktopDrive: Component<{Files: Accessor<Array<FileData>>; sortOptions: SelectOption[]; selectedSort: Accessor<string[]>; setSelectedSort: (value: string[]) => void; sortedFiles: Accessor<Array<FileData>>; searchQuery?: Accessor<string>; setSearch?: (v: string) => void}> = (props) => {
    // Lazy load files using IntersectionObserver
    const [visibleCount, setVisibleCount] = createSignal(50);
    let desktopScrollRef: HTMLDivElement | undefined;
    let desktopSentinelRef: HTMLDivElement | undefined;
    let desktopObserver: IntersectionObserver | undefined;
    const ctx = useContext(AppContext)!;

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
        desktopObserver = new IntersectionObserver(
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
            { root: desktopScrollRef, rootMargin: "0px 0px 200px 0px", threshold: 0 }
        );
        if (desktopSentinelRef) desktopObserver.observe(desktopSentinelRef);
    });

    onCleanup(() => desktopObserver?.disconnect());

    return (
        <DesktopTemplate CurrentPage="Files">
            <div class="flex flex-col w-full h-full px-[2vh] p-[1vh]">
                <div class="w-full flex justify-between items-center">
                    <p class="text-white font-black text-[4vh]">My Files</p>
                </div>
                <div class="flex width-full justify-end gap-3 h-12">
                    <div class="hidden md:block w-80">
                        <div class="relative">
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
                    <Show when={props.Files().length >= 2}>
                        <Select
                            options={props.sortOptions}
                            selected={props.selectedSort()}
                            onChange={(s) => props.setSelectedSort(s.length ? [s[s.length - 1]] : [])}
                            placeholderText="Sort By"
                            class="h-full"
                        />
                    </Show>
                    <UploadPopup />
                </div>
                <div class="h-5"/>
                <div ref={(el) => (desktopScrollRef = el)} class="w-full flex justify-center flex-wrap h-full gap-8 overflow-y-scroll custom-scrollbar">
                    <For each={displayedFiles()}  fallback={<FilesError />}>
                    {(file) => <FileCard File={file} />}
                    </For>
                    {/* Sentinel for lazy loading */}
                    <div ref={(el) => (desktopSentinelRef = el)} class="w-full h-px" />
                </div>
            </div>
        </DesktopTemplate>
    )
}

export default DesktopDrive;