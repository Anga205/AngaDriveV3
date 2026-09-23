import type { Component } from "solid-js"
import { For } from "solid-js"
import type { SelectableFile, FileUploadProgressData } from "../types";
import FileUploadPreview from "./FileUploadPreview";

interface UploadFileListProps {
    files: SelectableFile[];
    getUploadInfo: (uniqueId: string) => FileUploadProgressData | undefined;
    onDelete: (uniqueId: string) => void;
    canDelete: () => boolean;
}

const UploadFileList: Component<UploadFileListProps> = (props) => {
    return (
        <div class="w-full space-y-2 max-h-[50vh] overflow-y-auto custom-scrollbar p-1">
            <div class="w-full space-y-2">
                <For each={props.files}>
                    {(sf) => (
                        <FileUploadPreview
                            selectableFile={sf}
                            uploadInfo={() => props.getUploadInfo(sf.uniqueId)}
                            onDelete={props.onDelete}
                            canDelete={props.canDelete}
                        />
                    )}
                </For>
            </div>
        </div>
    );
};

export default UploadFileList;