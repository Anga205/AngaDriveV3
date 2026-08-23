import { Component, createMemo } from "solid-js";
import Dialog from '@corvu/dialog';
import { BinSVG } from "@/assets/SvgFiles";

interface BulkDeleteDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onDelete: () => void;
    selectedCount: number;
    triggerLabel?: string;
}

const BulkDeleteDialog: Component<BulkDeleteDialogProps> = (props) => {
    const title = createMemo(() =>
        `Delete ${props.selectedCount} selected file${props.selectedCount === 1 ? "" : "s"}?`
    );

    return (
        <Dialog open={props.open} onOpenChange={props.onOpenChange}>
            <Dialog.Trigger class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-red-600 hover:bg-red-700 text-white text-sm font-medium transition-colors duration-150">
                <BinSVG />
                {props.triggerLabel || `Delete ${props.selectedCount} File${props.selectedCount === 1 ? "" : "s"}`}
            </Dialog.Trigger>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
                <Dialog.Content class="flex z-50 justify-center flex-col fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-neutral-800 p-6 rounded-md shadow-lg text-white w-[clamp(300px,50vw,500px)]">
                    <Dialog.Label class="text-xl font-semibold mb-2 text-center">
                        {title()}
                    </Dialog.Label>
                    <p class="mb-4 text-sm text-neutral-400 text-center">
                        Once these files are deleted, they may not be recoverable again. Are you sure you want to permanently delete them?
                    </p>
                    <div class="flex justify-end space-x-3 mt-6">
                        <Dialog.Close class="bg-neutral-600 hover:bg-neutral-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200">
                            Cancel
                        </Dialog.Close>
                        <button
                            class="bg-red-600 hover:bg-red-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200"
                            onClick={props.onDelete}
                        >
                            Delete
                        </button>
                    </div>
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    );
};

export default BulkDeleteDialog;
