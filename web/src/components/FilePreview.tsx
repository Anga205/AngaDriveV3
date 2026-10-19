import type { FileData } from "../library/types"
import { FileTextSVG } from "../assets/SvgFiles";
import { AppContext } from "../Context";
import { createSignal, onCleanup, Component, Show, useContext } from "solid-js";
import { assetsUrl } from "@/assets/ApiUrl";

const PREVIEW_SIZE_LIMIT = 40 * 1024 * 1024; // 40 MB

const IMAGE_EXTENSIONS = ["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff", "heic", "heif"];
const VIDEO_EXTENSIONS = ["mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"];
const AUDIO_EXTENSIONS = ["mp3", "wav", "aac", "flac", "ogg", "wma", "m4a"];

const PreviewImage: Component<{ src: string }> = (props) => {
    const [loaded, setLoaded] = createSignal(false);
    const [failed, setFailed] = createSignal(false);
    return (
        <Show when={!failed()} fallback={<FileTextSVG class="max-h-full p-4 opacity-50" />}>
            <div class="relative flex items-center justify-center w-full h-full">
                <Show when={!loaded()}>
                    <FileTextSVG class="max-h-full p-4 opacity-50" />
                </Show>
                <img
                    src={props.src}
                    loading="lazy"
                    class="absolute inset-0 m-auto max-h-full max-w-full p-2"
                    onLoad={() => setLoaded(true)}
                    onError={() => setFailed(true)}
                />
            </div>
        </Show>
    );
};

// Video previews are generated asynchronously, so the endpoint answers 425
// (Too Early) until the WebP exists and 415 (Unsupported Media Type) when the
// source turned out to be corrupted or not a video. An <img> tag cannot see
// those statuses (it only fires onError), so fetch the preview explicitly and
// retry while it is being generated.
const VIDEO_PREVIEW_RETRY_MS = 2000;
const VIDEO_PREVIEW_MAX_ATTEMPTS = 30;

const VideoPreview: Component<{ src: string }> = (props) => {
    const [objectUrl, setObjectUrl] = createSignal<string>();
    const [unavailable, setUnavailable] = createSignal(false);
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;

    const load = async (attempt: number) => {
        try {
            const resp = await fetch(props.src);
            if (cancelled) return;

            if (resp.status === 425 && attempt < VIDEO_PREVIEW_MAX_ATTEMPTS) {
                timer = setTimeout(() => load(attempt + 1), VIDEO_PREVIEW_RETRY_MS);
                return;
            }
            if (!resp.ok) {
                setUnavailable(true);
                return;
            }

            const blob = await resp.blob();
            if (cancelled) return;
            setObjectUrl(URL.createObjectURL(blob));
        } catch {
            if (!cancelled) setUnavailable(true);
        }
    };

    load(0);

    onCleanup(() => {
        cancelled = true;
        if (timer) clearTimeout(timer);
        const url = objectUrl();
        if (url) URL.revokeObjectURL(url);
    });

    return (
        <Show when={!unavailable()} fallback={<FileTextSVG class="max-h-full p-4 opacity-50" />}>
            <Show when={objectUrl()} fallback={<FileTextSVG class="max-h-full p-4 opacity-50" />}>
                {(url) => (
                    <img
                        src={url()}
                        class="max-h-full max-w-full p-2"
                        onError={() => setUnavailable(true)}
                    />
                )}
            </Show>
        </Show>
    );
};

const FilePreview: Component<{ file: FileData }> = (props) => {
    const ctx = useContext(AppContext)!;
    const [isVisible, setIsVisible] = createSignal<boolean>(ctx.loadedFiles?.()?.has(props.file.file_directory) || false);
    let containerRef: HTMLDivElement | undefined;
    let observer: IntersectionObserver | undefined;

    // Find the nearest scrollable ancestor to use as the IntersectionObserver root
    const getScrollParent = (node: HTMLElement | null): HTMLElement | null => {
        let el: HTMLElement | null = node?.parentElement || null;
        while (el) {
            const style = getComputedStyle(el);
            const overflowY = style.overflowY;
            const overflow = style.overflow;
            const isScrollable = [overflowY, overflow].some((v) => v === "auto" || v === "scroll" || v === "overlay");
            if (isScrollable) return el;
            el = el.parentElement;
        }
        return null;
    };

    const isInView = (el: HTMLElement, rootEl: HTMLElement | null): boolean => {
        const rootRect = rootEl ? rootEl.getBoundingClientRect() : document.documentElement.getBoundingClientRect();
        const rect = el.getBoundingClientRect();
        return (
            rect.bottom > rootRect.top &&
            rect.top < rootRect.bottom &&
            rect.right > rootRect.left &&
            rect.left < rootRect.right
        );
    };

    const markLoaded = () => {
        try {
            ctx.setLoadedFiles?.((prev) => {
                const next = new Set(prev || new Set());
                next.add(props.file.file_directory);
                return next;
            });
        } catch (e) {
            // ignore if context not available
        }
    };

    const setRef = (el: HTMLDivElement) => {
        containerRef = el;
        if (!containerRef) return;

        const rootEl = getScrollParent(containerRef);

        // Create observer with the correct root (scroll container or viewport)
        observer = new IntersectionObserver(
            (entries) => {
                const entry = entries[0];
                if (entry.isIntersecting) {
                    setIsVisible(true);
                    // Persist that this file was loaded for the session
                    markLoaded();
                    observer?.unobserve(entry.target as Element);
                }
            },
            { root: rootEl, threshold: 0.01 }
        );

        // Defer observe to the next frame to avoid Chromium initial layout race
        requestAnimationFrame(() => {
            if (!containerRef) return;
            observer?.observe(containerRef);
        });

        // Fallback manual check (Chromium sometimes doesn't fire until scroll in nested scrollers)
        requestAnimationFrame(() => {
            if (!isVisible() && containerRef && isInView(containerRef, rootEl)) {
                setIsVisible(true);
                markLoaded();
                observer?.unobserve(containerRef);
            }
        });
    };

    onCleanup(() => {
        if (containerRef) {
            observer?.unobserve(containerRef);
        }
        observer?.disconnect();
        observer = undefined;
    });

    const PreviewContent: Component = () => {
        const ext = props.file.original_file_name.split('.').pop()?.toLowerCase();

        const isImage = IMAGE_EXTENSIONS.includes(ext || "");
        const isSvg = ext === "svg";
        const isVideo = VIDEO_EXTENSIONS.includes(ext || "");
        const isAudio = AUDIO_EXTENSIONS.includes(ext || "");
        const isPdf = ext === "pdf";

        // Non-SVG images always show their preview
        if (isImage) {
            return <PreviewImage src={assetsUrl(`/preview-image/${props.file.file_directory}`)} />;
        }

        // SVGs only preview when under the size limit
        if (isSvg && props.file.file_size <= PREVIEW_SIZE_LIMIT) {
            return <PreviewImage src={assetsUrl(`/preview-image/${props.file.file_directory}`)} />;
        }

        if (isVideo) {
            return <VideoPreview src={assetsUrl(`/preview-video/${props.file.file_directory}.webp`)} />;
        }
        if (isAudio && props.file.file_size <= PREVIEW_SIZE_LIMIT) {
            return <audio src={assetsUrl(`/i/${props.file.file_directory}`)} controls class="w-full" />;
        }
        if (isPdf) {
            return <PreviewImage src={assetsUrl(`/preview/${props.file.file_directory}.jpg`)} />;
        }
        return <FileTextSVG class="max-h-full p-4 opacity-50" />;
    };

    return (
        <div ref={setRef} class="flex justify-center items-center w-full h-full opacity-70">
            <Show when={isVisible()} fallback={<FileTextSVG class="max-h-full p-4 opacity-50" />}>
                <PreviewContent />
            </Show>
        </div>
    );
};

export default FilePreview;