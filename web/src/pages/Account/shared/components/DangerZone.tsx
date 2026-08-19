import { Component, createSignal } from "solid-js";
import Dialog from '@corvu/dialog';
import { toast } from "solid-toast";
import { useWebSocket } from "@/Websockets";

const DeleteAccountDialog: Component = () => {
    const [open, setOpen] = createSignal(false);
    const [deletePassword, setDeletePassword] = createSignal("");
    const { socket: getSocket, status: socketStatus } = useWebSocket();
    const handleAccountDeletion = () => {
        if (socketStatus() !== "connected") {
            setTimeout(() => {
                handleAccountDeletion();
            }, 100);
            return;
        }
        if (!deletePassword()) {
            toast.error("Please enter your password to confirm account deletion.");
            return;
        }
        const currentSocket = getSocket();
        currentSocket?.send(
            JSON.stringify({
                type: "delete_account",
                data: {
                    email: localStorage.getItem("email"),
                    password: deletePassword(),
                },
            })
        )
    }
    return (
        <Dialog open={open()} onOpenChange={(newOpen) => {
            setOpen(newOpen);
            setDeletePassword("");
        }}>
            <Dialog.Trigger class="bg-red-600 w-full hover:bg-red-700 text-white font-semibold py-[1vh] px-[1vw] rounded mt-auto transition-colors duration-200 text-[1.5vh]">
                Delete&nbsp;Account
            </Dialog.Trigger>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
                <Dialog.Content class="flex z-50 justify-center flex-col fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-neutral-800 p-6 rounded-md shadow-lg text-white w-[clamp(300px,50vw,500px)]">
                    <Dialog.Label class="text-xl font-semibold mb-2 text-center">Confirm Account Deletion</Dialog.Label>
                    <p class="mb-4 text-sm text-neutral-400">
                        P.S. if u delete ur account then i cant recover it even if you ask me to, all ur files, collections, and data will be instantly deleted automatically
                    </p>
                    <label for="deletePasswordInput" class="text-sm text-red-400 mb-1 font-semibold">
                        Enter ur password to confirm:
                    </label>
                    <input
                        id="deletePasswordInput"
                        type="password"
                        class="w-full p-3 rounded bg-neutral-700 text-[1.5vh] placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-red-400"
                        placeholder="Your Password"
                        autocomplete="current-password"
                        onInput={e => setDeletePassword(e.currentTarget.value)}
                    />
                    <div class="flex justify-end space-x-3 mt-6">
                        <Dialog.Close class="bg-neutral-600 hover:bg-neutral-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200">
                            Cancel
                        </Dialog.Close>
                        <button
                            class="bg-red-600 hover:bg-red-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200"
                            onClick={handleAccountDeletion}
                        >
                            Delete Account
                        </button>
                    </div>
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    )
}

const DangerZone: Component<{logout: () => void; class?: string}> = (props) => {
    return (
        <div class={`bg-red-950 border border-red-700 rounded-md p-[2vh] text-white flex flex-col shadow-md w-full ${props.class}`}>
            <p class="font-semibold text-[2vh] mb-[2vh] text-center">Danger Zone</p>
            <div class="w-full flex space-x-[1vh]">
                <DeleteAccountDialog/>
                <button class="bg-yellow-600 w-full hover:bg-yellow-700 text-white font-semibold py-[1vh] px-[1vw] rounded mt-auto transition-colors duration-200 text-[1.5vh]"
                onClick={() => {
                    props.logout();
                }}
                >
                    Log Out
                </button>
            </div>
        </div>
    )
}

const UserStat: Component<{title: string; value: string; class?: string}> = (props) => {
    return (
        <div class={`bg-neutral-900 w-full h-full rounded-md border border-neutral-700 flex justify-between items-center flex-col text-white p-[1vh] ${props.class ?? ''}`}>
            <p class="text-blue-700 font-bold text-center text-[2.5vh]">{props.title}</p>
            <p class="text-[2vh]">{props.value}</p>
        </div>
    )
}

export { DangerZone, UserStat };
