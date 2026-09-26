import { Component, createMemo } from "solid-js";
import Dialog from '@corvu/dialog';
import Tooltip from "@corvu/tooltip";
import { StickyNotes } from "@/assets/SvgFiles";

interface BulkDuplicateDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onDuplicate: () => void;
    selectedCount: number;
}

const BulkDuplicateDialog: Component<BulkDuplicateDialogProps> = (props) => {
    const title = createMemo(() =>
        `Duplicate ${props.selectedCount} selected file${props.selectedCount === 1 ? "" : "s"}?`
    );

    return (
        <Dialog open={props.open} onOpenChange={props.onOpenChange}>
            <Tooltip placement="bottom" openDelay={0} closeDelay={0}>
                <Tooltip.Trigger
                    as={Dialog.Trigger}
                    class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-neutral-900 hover:bg-neutral-950 text-white text-sm font-medium transition-colors duration-150"
                    aria-label="Duplicate selected files"
                >
                    <StickyNotes class="h-4 w-4" />
                </Tooltip.Trigger>
                <Tooltip.Content class="bg-neutral-900 text-white px-2 py-1 rounded z-50">
                    Duplicate&nbsp;Selected
                </Tooltip.Content>
            </Tooltip>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
                <Dialog.Content class="flex z-50 justify-center flex-col fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-neutral-800 p-6 rounded-md shadow-lg text-white w-[clamp(300px,50vw,500px)]">
                    <Dialog.Label class="text-xl font-semibold mb-2 text-center">
                        {title()}
                    </Dialog.Label>
                    <p class="mb-4 text-sm text-neutral-400 text-center">
                        This will create a copy of each selected file using the existing “Copy of …” naming pattern. Continue?
                    </p>
                    <div class="flex justify-end space-x-3 mt-6">
                        <Dialog.Close class="bg-neutral-600 hover:bg-neutral-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200">
                            Cancel
                        </Dialog.Close>
                        <button
                            class="bg-green-600 hover:bg-green-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200"
                            onClick={props.onDuplicate}
                        >
                            Duplicate
                        </button>
                    </div>
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    );
};

export default BulkDuplicateDialog;
