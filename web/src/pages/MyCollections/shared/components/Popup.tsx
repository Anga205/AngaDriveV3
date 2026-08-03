import { Accessor, createEffect, createSignal } from 'solid-js';
import Dialog from '@corvu/dialog';
import { useWebSocket } from '@/Websockets';
import toast from 'solid-toast';

const Popup = () => {
    const [newCollectionName, setNewCollectionName] = createSignal<string>('');
    const [modifying, setModifying] = createSignal<null | "Github" | "New">(null);
    const [githubURL, setGithubURL] = createSignal<string>('');
    createEffect(()=>{
        if (newCollectionName() !== '') {
            setModifying("New");
        } else if (githubURL() !== '') {
            setModifying("Github");
        } else {
            setModifying(null);
        }
    })
    const { socket: getSocket, status } = useWebSocket();
    const onSubmit = () => {
        if (status() !== "connected") {
            toast.error("Could not secure a connection to the server. Please try again later.");
            return;
        }
        const socket = getSocket()!;
        if (modifying() === "New") {
            let name = newCollectionName().trim();
            setNewCollectionName('');
            socket.send(JSON.stringify({
                type: "new_collection",
                data: {
                    collection_name: name,
                    auth: {
                        token: localStorage.getItem('token') || '',
                        email: localStorage.getItem('email') || '',
                        password: localStorage.getItem('password') || ''
                    }
                }
            }))
        } else if (modifying() === "Github") {
            let url = githubURL().trim();
            setGithubURL('');
            socket.send(JSON.stringify({
                type: "import_from_github",
                data: {
                    repo_url: url,
                    auth: {
                        token: localStorage.getItem('token') || '',
                        email: localStorage.getItem('email') || '',
                        password: localStorage.getItem('password') || ''
                    }
                }
            }));
        }
    }
    const disableSubmitButton: Accessor<boolean> = () => {
        if (modifying() === "New") {
            return newCollectionName().trim().length <= 2;
        } else if (modifying() === "Github") {
            const githubRepoRegex = /^https:\/\/github\.com\/[a-zA-Z0-9_-]+\/[a-zA-Z0-9_-]+$/;
            return !githubRepoRegex.test(githubURL());
        }
        return false;
    }
    return (
        <Dialog onOpenChange={() => {
            setNewCollectionName('');
            setGithubURL('');
            setModifying(null);
        }}>
            <Dialog.Trigger class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-green-600 hover:bg-green-800 py-[0.2vh] px-[1vh] rounded-[1vh] font-bold md:translate-y-[4vh]">
                <span class="text-4xl text-center">+</span>&nbsp;Create&nbsp;Collection
            </Dialog.Trigger>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 z-50 bg-black/50 data-open:animate-in data-open:fade-in-0% data-closed:animate-out data-closed:fade-out-0%"/>
                <Dialog.Content class="fixed z-50 top-[50%] left-[50%] translate-x-[-50%] translate-y-[-50%] w-[90vw] max-w-md bg-neutral-800 rounded-lg p-6 space-y-4">
                    <p class="text-white text-lg font-bold mb-4 text-center">Create Collection</p>
                    {(modifying() === "New" || modifying() === null) && (
                        <input type="text" placeholder="Collection Name" onInput={(e) => setNewCollectionName(e.target.value)} class="w-full p-2 rounded-lg bg-neutral-700 text-white focus:outline-none focus:ring-2 focus:ring-green-500"/>
                    )}
                    {modifying() === null && (
                        <div class="flex w-full items-center justify-center">
                            <hr class="w-full border-neutral-600"/>
                            <p class="mx-2 text-gray-500">OR</p>
                            <hr class="w-full border-neutral-600"/>
                        </div>
                    )}
                    {(modifying() === null || modifying() === "Github") && (
                        <input type="text" placeholder="Import a GitHub Repository" onInput={(e) => setGithubURL(e.target.value)} class="w-full p-2 rounded-lg bg-neutral-700 text-white focus:outline-none focus:ring-2 focus:ring-green-500"/>
                    )}
                    {(modifying() === "New" || modifying() === "Github") && (
                        <Dialog.Close
                            onClick={onSubmit}
                            class={`bg-green-700 disabled:bg-neutral-600 text-white p-2 rounded-lg font-semibold w-full hover:bg-green-800 transition-colors ${modifying() === "New" && newCollectionName().trim().length <= 2 ? 'bg-gray-500 cursor-not-allowed' : 'hover:bg-green-800'}`}
                            disabled={disableSubmitButton()}
                        >
                            {modifying() === "New" ? "Create" : "Import"}
                        </Dialog.Close>
                    )}
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    )
}

export default Popup;