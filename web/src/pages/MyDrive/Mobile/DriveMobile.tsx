import { For, Show, Accessor, Component, createEffect, createMemo, createSignal, onCleanup, onMount, useContext } from "solid-js";
import { FileData } from "../../../library/types";
import Select, { SelectOption } from "../../../components/Select";
import { AppContext } from "../../../Context";
import Navbar from "../../../components/Navbar";
import Search from "lucide-solid/icons/search";
import { UploadPopup } from "../shared/components/UploadPopUp";
import FilesError from "../shared/components/FilesError";
import FileCard from "../../../components/FileCard";

const MobileDrive: Component<{Files: Accessor<Array<FileData>>; sortOptions: SelectOption[]; selectedSort: Accessor<string[]>; setSelectedSort: (value: string[]) => void; sortedFiles: () => Array<FileData>; searchQuery?: Accessor<string>; setSearch?: (v: string) => void}> = (props) => {
    // Lazy load for mobile as well
    const [visibleCount, setVisibleCount] = createSignal(50);
    let mobileScrollRef: HTMLDivElement | undefined;
    let mobileSentinelRef: HTMLDivElement | undefined;
    let mobileObserver: IntersectionObserver | undefined;

    const ctx = useContext(AppContext)!;

    const displayedFiles = createMemo(() => {
        const base = props.sortedFiles() || [];
        const visible = base.slice(0, visibleCount());
        const loaded = Array.from(ctx.loadedFiles?.() || new Set<string>());
        const extras = base.filter(f => loaded.includes(f.file_directory) && !visible.some(v => v.file_directory === f.file_directory));
        return [...visible, ...extras];
    });

    createEffect(() => {
        const total = props.sortedFiles().length;
        setVisibleCount(Math.min(50, total));
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
            </div>
            <div ref={(el) => (mobileScrollRef = el)} class="w-full px-4 mt-4 max-h-full h-full flex flex-wrap items-center space-y-4 space-x-4 justify-center overflow-y-auto">
                <For each={displayedFiles()} fallback={<FilesError />}>
                    {(file) => (
                        <FileCard File={file} />
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