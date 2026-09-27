import { Component, createSignal, Show } from "solid-js";
import Tooltip from "@corvu/tooltip";
import { CrossSVG } from "@/assets/SvgFiles";
import BulkDeleteDialog from "./BulkDeleteDialog";
import BulkDuplicateDialog from "./BulkDuplicateDialog";
import BulkAddToCollectionDialog from "./BulkAddToCollectionDialog";

interface BulkActionsMenuProps {
    selectedCount: number;
    onUnselectAll: () => void;
    onDelete: () => void;
    onDuplicate: () => void;
    onAddToCollection: (collectionId: string, collectionName: string) => void;
}

const BulkActionsMenu: Component<BulkActionsMenuProps> = (props) => {
    const [deleteOpen, setDeleteOpen] = createSignal(false);
    const [duplicateOpen, setDuplicateOpen] = createSignal(false);
    const [addToCollectionOpen, setAddToCollectionOpen] = createSignal(false);

    return (
        <Show when={props.selectedCount > 0}>
            <div class="flex gap-3">
                <Tooltip placement="bottom" openDelay={0} closeDelay={0}>
                    <Tooltip.Trigger
                        class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-neutral-800 md:bg-neutral-900 hover:bg-neutral-950 text-neutral-200 text-sm font-medium transition-colors duration-150"
                        onClick={props.onUnselectAll}
                        aria-label="Clear selection"
                    >
                        <CrossSVG />
                    </Tooltip.Trigger>
                    <Tooltip.Content class="bg-neutral-900 text-white px-2 py-1 rounded z-50">
                        Clear&nbsp;Selection
                    </Tooltip.Content>
                </Tooltip>
                <BulkAddToCollectionDialog
                    open={addToCollectionOpen()}
                    onOpenChange={setAddToCollectionOpen}
                    onAddToCollection={props.onAddToCollection}
                    selectedCount={props.selectedCount}
                />
                <BulkDuplicateDialog
                    open={duplicateOpen()}
                    onOpenChange={setDuplicateOpen}
                    onDuplicate={props.onDuplicate}
                    selectedCount={props.selectedCount}
                />
                <BulkDeleteDialog
                    open={deleteOpen()}
                    onOpenChange={setDeleteOpen}
                    onDelete={props.onDelete}
                    selectedCount={props.selectedCount}
                />
            </div>
        </Show>
    );
};

export default BulkActionsMenu;