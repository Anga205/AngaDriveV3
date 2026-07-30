import type { Component } from "solid-js"
import { createSignal, createMemo, onCleanup, createEffect, useContext } from "solid-js"
import { useWebSocket } from "@/Websockets";
import { FileData } from "@/library/types";
import { Toaster, toast } from 'solid-toast';
import { AppContext } from "@/Context";
import DesktopDrive from "./Desktop/DriveDesktop";
import MobileDrive from "./Mobile/DriveMobile";

const MyDrive: Component = () => {
    const [isMobile, setIsMobile] = createSignal(window.innerWidth <= 768);

    const handleResize = () => {
        setIsMobile(window.innerWidth <= 768);
    };
    window.addEventListener('resize', handleResize);
    onCleanup(() => window.removeEventListener('resize', handleResize));

    const ctx = useContext(AppContext)!;

    const { socket: getSocket } = useWebSocket();

    const messageHandler = (event: MessageEvent) => {
        const data = JSON.parse(event.data);
        if (data.type === "convert_video_response") {
            if (data.data.error) {
                toast.error(`${data.data.error}`);
            } else if (data.data.file) {
                toast.success(`Video converted successfully: ${data.data.file.original_file_name}`);
            }
        } else if (data.type === "delete_file_response") {
            if (data.data.error) {
                toast.error(`Error deleting file: ${data.data.error}`);
            } else {
                toast.success(`File deleted successfully: ${data.data.success}`);
            }
        }
    }


    createEffect(() => {
        const socket = getSocket();
        if (!socket) return;
        socket.addEventListener("message", messageHandler);
        onCleanup(() => {
            socket.removeEventListener("message", messageHandler);
        });
    })

    const sortOptions = [
        { id: 'time_desc', name: 'Time: Newest first' },
        { id: 'time_asc', name: 'Time: Old first' },
        { id: 'name_asc', name: 'Name: Alphabetical' },
        { id: 'name_desc', name: 'Name: Reverse alphabetical' },
        { id: 'size_desc', name: 'Size: Largest first' },
        { id: 'size_asc', name: 'Size: Smallest first' },
    ];

    // Sorting state and options
    const [selectedSort, setSelectedSort] = createSignal<string[]>(['time_desc']);

    // Live search state
    const [searchQuery, setSearchQuery] = createSignal<string>('');

    const sortedFiles = createMemo(() => {
        const files = ctx.files() || [];
        const sel = selectedSort()[0] || 'time_desc';
        const arr = files.slice();

        // Reusable comparator helpers
        const byTimeAsc = (a: FileData, b: FileData) => a.timestamp - b.timestamp;
        const byTimeDesc = (a: FileData, b: FileData) => b.timestamp - a.timestamp;
        const byNameAsc = (a: FileData, b: FileData) => a.original_file_name.localeCompare(b.original_file_name, undefined, { sensitivity: 'variant' });
        const byNameDesc = (a: FileData, b: FileData) => b.original_file_name.localeCompare(a.original_file_name, undefined, { sensitivity: 'variant' });
        const bySizeAsc = (a: FileData, b: FileData) => a.file_size - b.file_size;
        const bySizeDesc = (a: FileData, b: FileData) => b.file_size - a.file_size;
        const byPathAsc = (a: FileData, b: FileData) => a.file_directory.localeCompare(b.file_directory);

        const chain = (...comparators: Array<(a: FileData, b: FileData) => number>) =>
            (a: FileData, b: FileData) => {
                for (const cmp of comparators) {
                    const res = cmp(a, b);
                    if (res !== 0) return res;
                }
                return 0;
            };

        const comparatorMap: Record<string, (a: FileData, b: FileData) => number> = {
            time_desc: chain(byTimeDesc, byPathAsc),
            time_asc: chain(byTimeAsc, byPathAsc),
            name_asc: chain(byNameAsc, byTimeAsc, byPathAsc),
            name_desc: chain(byNameDesc, byTimeDesc, byPathAsc),
            size_desc: chain(bySizeDesc, byTimeDesc, byPathAsc),
            size_asc: chain(bySizeAsc, byTimeAsc, byPathAsc),
        };

        return arr.sort(comparatorMap[sel] || comparatorMap.time_desc);
    });

    // Filtered files based on live search (name or directory)
    const filteredFiles = createMemo(() => {
        const q = searchQuery().trim().toLowerCase();
        if (!q) return sortedFiles();
        return sortedFiles().filter(f => (f.original_file_name || '').toLowerCase().includes(q) || (f.file_directory || '').toLowerCase().includes(q));
    });

    // Clear persisted loaded files whenever the user changes search or sort
    createEffect(() => {
        // depend on selectedSort and searchQuery
        selectedSort();
        searchQuery();
        try {
            ctx.setLoadedFiles?.(new Set());
        } catch (e) {}
    });

    return (
        <>
            <title>My Files | DriveV3</title>
            {isMobile() ? <MobileDrive Files={ctx.files} sortOptions={sortOptions} selectedSort={selectedSort} setSelectedSort={setSelectedSort} sortedFiles={filteredFiles} searchQuery={searchQuery} setSearch={setSearchQuery}/> : <DesktopDrive Files={ctx.files} sortOptions={sortOptions} selectedSort={selectedSort} setSelectedSort={setSelectedSort} sortedFiles={filteredFiles} searchQuery={searchQuery} setSearch={setSearchQuery}/>} 
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
    )
}

export { MyDrive }