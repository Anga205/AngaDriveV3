import type { Component } from "solid-js";
import { createSignal } from "solid-js";
import { useLocation, useNavigate } from "@solidjs/router";
import Navbar from "./Navbar";
import type { Pages } from "../library/types";

const DesktopTemplate: Component<{ CurrentPage: Pages; children: any }> = (props) => {
    const navigate = useNavigate();
    const location = useLocation();
    const [isDragOver, setIsDragOver] = createSignal(false);

    const handleDrop = (e: DragEvent) => {
        e.preventDefault();
        setIsDragOver(false);
        const dt = e.dataTransfer;
        if (!dt || !dt.files || dt.files.length === 0) return;
    const files = Array.from(dt.files);

        const path = location.pathname || "";
        if (path.startsWith("/collection")) {
            // Same page: just dispatch event, don't stash pending to avoid duplicates
            document.dispatchEvent(new CustomEvent("open-collection-upload", { detail: { files } }));
            return;
        }

        const openDriveUpload = () => {
            document.dispatchEvent(new CustomEvent("open-drive-upload", { detail: { files } }));
        };

        if (path !== "/my_drive") {
            // Navigation case: stash pending and navigate. Target page will consume and open.
            (window as any).__pendingDriveUploadFiles = files;
            navigate("/my_drive");
        } else {
            openDriveUpload();
        }
    };

    return (
        <div
            class="max-h-screen w-screen h-screen flex items-start bg-black overflow-hidden"
            onDragOver={(e) => { e.preventDefault(); if (!isDragOver()) setIsDragOver(true); }}
            onDragEnter={(e) => { e.preventDefault(); if (!isDragOver()) setIsDragOver(true); }}
            onDragLeave={() => setIsDragOver(false)}
            onDrop={handleDrop}
        >
            <Navbar CurrentPage={props.CurrentPage} Type="desktop"/>
            <div class="w-[calc(100vw-103px)] h-full">
                {props.children}
            </div>
            {/* Optional subtle overlay to indicate drop is allowed */}
            {isDragOver() && (
                <div class="pointer-events-none fixed inset-0 border-2 border-dashed border-blue-700/90"/>
            )}
        </div>
    );
}

export {DesktopTemplate}