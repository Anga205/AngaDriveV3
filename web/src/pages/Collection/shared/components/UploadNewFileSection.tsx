import type { Component } from "solid-js";
import type { SelectableFile, FileUploadProgressData } from "@/pages/MyDrive/shared/types";
import UploadFileList from "@/pages/MyDrive/shared/components/UploadFileList";

interface UploadNewFileSectionProps {
    files: SelectableFile[];
    isDragOver: boolean;
    flashPaste: boolean;
    getUploadInfo: (uniqueId: string) => FileUploadProgressData | undefined;
    onDelete: (uniqueId: string) => void;
    canDelete: () => boolean;
    onDragOver: (e: DragEvent) => void;
    onDragEnter: (e: DragEvent) => void;
    onDragLeave: () => void;
    onDrop: (e: DragEvent) => void;
    onFileChange: (e: Event) => void;
}

const UploadNewFileSection: Component<UploadNewFileSectionProps> = (props) => {
    return (
        <label
            for="collection-file-upload"
            class={`rounded-md min-h-[15vh] flex justify-center items-center cursor-pointer my-[1vh] ${props.files.length === 0 ? `border-2 ${props.isDragOver ? 'border-blue-400' : 'border-dotted border-blue-800'}` : ''}${props.flashPaste ? ' ring-2 ring-blue-500 animate-pulse' : ''}`}
            onDragOver={props.onDragOver}
            onDragEnter={props.onDragEnter}
            onDragLeave={props.onDragLeave}
            onDrop={props.onDrop}
        >
            {props.files.length === 0 ? (
                <p class="text-center p-4 text-white">Drag and drop files here or click to select files</p>
            ) : (
                <UploadFileList
                    files={props.files}
                    getUploadInfo={props.getUploadInfo}
                    onDelete={props.onDelete}
                    canDelete={props.canDelete}
                />
            )}
            <input id="collection-file-upload" type="file" multiple class="hidden" onChange={props.onFileChange} />
        </label>
    );
};

export default UploadNewFileSection;