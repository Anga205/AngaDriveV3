import { Accessor, Component, createMemo, createSignal, onCleanup, Show } from "solid-js";
import { BinSVG, FileSVG } from "@/assets/SvgFiles";
import { formatFileSize, truncateFileName } from "@/library/functions";
import { FileUploadProgressData, SelectableFile } from "../types";

const FileUploadPreview: Component<{
    selectableFile: SelectableFile;
    uploadInfo?: Accessor<FileUploadProgressData | undefined>;
    onDelete: (uniqueId: string) => void;
    canDelete?: Accessor<boolean>;
}> = (props) => {

    const preview_size_limit = 100 * 1024 * 1024; // 100 MB
    const ext = props.selectableFile.file.name.split('.').pop()?.toLowerCase();
    const file = props.selectableFile.file;

    const info = createMemo(() => props.uploadInfo ? props.uploadInfo() : undefined);
    const [objectUrl, setObjectUrl] = createSignal<string | undefined>();

    createMemo(() => {
        if (ext && (["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"].includes(ext) || ["mp4", "mkv", "mov", "wmv", "flv", "webm"].includes(ext)) && file.size < preview_size_limit) {
            const url = URL.createObjectURL(file);
            setObjectUrl(url);
            onCleanup(() => URL.revokeObjectURL(url));
        } else {
            setObjectUrl(undefined);
        }
    });

    return (
        <div class="flex flex-col sm:flex-row items-center justify-between p-3 bg-neutral-800 rounded-lg shadow-lg hover:shadow-xl transition-shadow w-full sm:w-auto min-h-20 sm:min-h-16 overflow-hidden">
            <div class="flex items-center space-x-4 w-full sm:w-auto">
                <div class="w-12 h-12 flex items-center justify-center rounded-md overflow-hidden">
                    {(() => {
                        const url = objectUrl();
                        if (ext && ["pdf"].includes(ext)) {
                            return <p class="text-white text-xs">PDF</p>;
                        }
                        if (url && ext && ["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"].includes(ext)) {
                            return <img src={url} alt="Preview" class="w-full h-full object-cover" />;
                        }
                        if (url && ext && ["mp4", "mkv", "mov", "wmv", "flv", "webm"].includes(ext)) {
                            return <video src={url} class="w-full h-full object-cover" muted />;
                        }
                        return <FileSVG />;
                    })()}
                </div>
                <div class="flex flex-col w-full sm:w-auto">
                    <p class="text-white text-sm font-medium truncate max-w-50">{truncateFileName(file.name)}</p>
                    <p class="text-neutral-400 text-xs">{formatFileSize(file.size)}</p>
                </div>
            </div>
            <div class="grow w-full sm:w-auto mt-3 sm:mt-0 sm:ml-4 min-w-25">
                <Show when={info() && (info()!.status === 'uploading' || info()!.status === 'pending')}>
                    <div class="w-full bg-neutral-700 rounded-full h-2.5">
                        <div
                            class="bg-blue-600 h-2.5 rounded-full transition-all duration-100 ease-linear"
                            style={{ width: `${info() ? info()!.progress : 0}%` }}
                        ></div>
                    </div>
                    <p class="text-xs text-neutral-400 mt-1 text-right sm:text-left">{info()!.status === 'pending' ? 'Pending...' : `${info()!.progress}%`}</p>
                </Show>
                <Show when={info() && info()!.status === 'processing'}>
                    <div class="w-full bg-neutral-700 rounded-full h-2.5">
                        <div
                            class="bg-green-600 h-2.5 rounded-full transition-all duration-100 ease-linear"
                            style={{ width: `100%` }}
                        ></div>
                    </div>
                    <p class="text-xs text-blue-400 mt-1 text-right sm:text-left" title="Upload complete — processing on server. You may close this window.">
                        100% Uploaded — Processing on server...
                    </p>
                </Show>
                <Show when={info() && info()!.status === 'completed'}>
                    <p class="text-sm font-semibold text-green-500 text-right sm:text-left md:text-right">Uploaded!</p>
                </Show>
                <Show when={info() && info()!.status === 'error'}>
                    <p class="text-sm font-semibold text-red-500 text-right sm:text-left" title={info()!.errorMessage}>Error</p>
                </Show>
            </div>
            <button
                class="flex items-center justify-center p-2 mt-3 sm:mt-0 sm:ml-2 bg-red-700/30 hover:bg-red-700/20 rounded-lg text-red-500 hover:text-red-700 transition-colors w-full sm:w-auto h-10"
                onClick={() => props.onDelete(props.selectableFile.uniqueId)}
                disabled={props.canDelete ? !props.canDelete() : (!!info() && info()!.status === 'uploading')}
            >
                <BinSVG />
            </button>
        </div>
    )
}

export default FileUploadPreview;