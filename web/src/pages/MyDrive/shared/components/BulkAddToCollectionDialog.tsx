import { Component, createMemo, createSignal } from "solid-js";
import Dialog from '@corvu/dialog';
import Tooltip from "@corvu/tooltip";
import { FolderPlusSVG } from "@/assets/SvgFiles";
import Select from "@/components/Select";
import { useContext } from "solid-js";
import { AppContext } from "@/Context";
import { Show } from "solid-js";

interface BulkAddToCollectionDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onAddToCollection: (collectionId: string, collectionName: string) => void;
    selectedCount: number;
}

const BulkAddToCollectionDialog: Component<BulkAddToCollectionDialogProps> = (props) => {
    const ctx = useContext(AppContext)!;
    const [collectionId, setCollectionId] = createSignal<string[]>([]);
    const [newCollectionName, setNewCollectionName] = createSignal("");

    const collectionOptions = createMemo(() => {
        return Array.from(ctx.userCollections?.() || new Set<string>())
            .map((id) => ctx.knownCollectionCards?.()[id])
            .filter((collection): collection is NonNullable<typeof collection> => !!collection)
            .map((collection) => ({ id: collection.id, name: collection.name }));
    });

    const hasCollections = createMemo(() => (collectionOptions().length > 0));

    const handleConfirm = () => {
        const selectedCollectionId = collectionId()[0] || "";
        if (selectedCollectionId) {
            props.onAddToCollection(selectedCollectionId, "");
            props.onOpenChange(false);
            return;
        }
        const name = newCollectionName().trim();
        if (!name) return;
        props.onAddToCollection("", name);
        props.onOpenChange(false);
    };

    const reset = () => {
        setCollectionId([]);
        setNewCollectionName("");
    };

    return (
        <Dialog open={props.open} onOpenChange={(open) => {
            props.onOpenChange(open);
            if (!open) reset();
        }}>
            <Tooltip placement="bottom" openDelay={0} closeDelay={0}>
                <Tooltip.Trigger
                    as={Dialog.Trigger}
                    class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-neutral-800 md:bg-neutral-900 hover:bg-neutral-950 text-white text-sm font-medium transition-colors duration-150"
                    aria-label="Add selected files to a collection"
                >
                    <FolderPlusSVG class="w-4 h-4" />
                </Tooltip.Trigger>
                <Tooltip.Content class="bg-neutral-900 text-white px-2 py-1 rounded z-50">
                    Add&nbsp;to&nbsp;Collection
                </Tooltip.Content>
            </Tooltip>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
                <Dialog.Content class="flex z-50 flex-col fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-neutral-800 p-6 rounded-md shadow-lg text-white w-[clamp(320px,50vw,500px)]">
                    <Dialog.Label class="text-xl font-semibold mb-4 text-center">
                        Add {props.selectedCount} selected file{props.selectedCount === 1 ? "" : "s"} to a collection
                    </Dialog.Label>
                        <div class="space-y-4">
                            <Show when={hasCollections() && newCollectionName() === ""}>
                                <Select
                                    options={collectionOptions()}
                                    selected={collectionId()}
                                    onChange={setCollectionId}
                                    placeholderText="Choose an existing collection"
                                />
                            </Show>
                            <Show when={hasCollections() && (newCollectionName() === "") && (collectionId().length === 0)}>
                                <div class="flex w-full items-center justify-center">
                                    <hr class="w-full border-neutral-600" />
                                    <p class="mx-2 text-gray-500">OR</p>
                                    <hr class="w-full border-neutral-600" />
                                </div>
                            </Show>
                            <Show when={collectionId().length === 0}>
                                <input
                                    type="text"
                                    value={newCollectionName()}
                                    onInput={(e) => setNewCollectionName((e.target as HTMLInputElement).value)}
                                    placeholder="Collection name"
                                    class="w-full p-2 rounded-lg bg-neutral-700 text-white focus:outline-none focus:ring-2 focus:ring-green-500"
                                />
                            </Show>
                            <Show when={collectionId().length > 0 || newCollectionName().trim().length > 0}>
                                <div class="flex justify-between space-x-3 pt-2">
                                    <Show when={collectionId().length > 0} fallback={<div />}>
                                        <button
                                            type="button"
                                            class="bg-neutral-600 hover:bg-neutral-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200"
                                            onClick={() => setCollectionId([])}
                                        >
                                            Clear&nbsp;Selection
                                        </button>
                                    </Show>
                                    <div class="flex justify-end space-x-3">
                                        <Show when={collectionId().length > 0} fallback={
                                            <button
                                                type="button"
                                                onClick={handleConfirm}
                                                disabled={newCollectionName().trim().length < 1}
                                                class="bg-green-600 hover:bg-green-700 disabled:bg-neutral-600 disabled:cursor-not-allowed text-white font-semibold py-2 px-4 rounded transition-colors duration-200"
                                            >
                                                Create & Add
                                            </button>
                                        }>
                                            <button
                                                type="button"
                                                onClick={handleConfirm}
                                                disabled={collectionId().length === 0}
                                                class="bg-green-600 hover:bg-green-700 disabled:bg-neutral-600 disabled:cursor-not-allowed text-white font-semibold py-2 px-4 rounded transition-colors duration-200"
                                            >
                                                Add
                                            </button>
                                        </Show>
                                    </div>
                                </div>
                            </Show>
                        </div>
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    );
};

export default BulkAddToCollectionDialog;
