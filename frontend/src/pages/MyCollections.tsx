import { Component, createEffect, createSignal, useContext } from 'solid-js';
import { DesktopTemplate } from '../components/Template';
import Navbar from '../components/Navbar';
import Dialog from '@corvu/dialog';
import { useWebSocket } from '../Websockets';
import toast, { Toaster } from 'solid-toast';
import { For } from 'solid-js';
import { AppContext } from '../Context';
import { ErrorSVG, InfoSVG } from '../assets/SvgFiles';
import CollectionCard from '../components/CollectionCard';

const Popup = () => {
    const [newCollectionName, setNewCollectionName] = createSignal<string>('');
    const [modifying, setModifying] = createSignal<null | "Github" | "New">(null);
    createEffect(()=>{
        if (newCollectionName() !== '') {
            setModifying("New");
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
        }
    }
    return (
        <Dialog onOpenChange={() => {}}>
            <Dialog.Trigger class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-green-600 hover:bg-green-800 md:p-[0.2vh] px-[1vh] rounded-[1vh] font-bold md:translate-y-[4vh]">
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
                        <input type="text" disabled={true} placeholder="Import a GitHub Repository" class="cursor-not-allowed w-full p-2 rounded-lg bg-neutral-700 text-white focus:outline-none focus:ring-2 focus:ring-green-500"/>
                    )}
                    {(modifying() === "New" || modifying() === "Github") && (
                        <Dialog.Close 
                            onClick={onSubmit} 
                            class={`bg-green-700 disabled:bg-neutral-600 text-white p-2 rounded-lg font-semibold w-full hover:bg-green-800 transition-colors ${modifying() === "New" && newCollectionName().trim().length <= 2 ? 'bg-gray-500 cursor-not-allowed' : 'hover:bg-green-800'}`}
                            disabled={modifying() === "New" && newCollectionName().trim().length <= 2}
                        >
                            {modifying() === "New" ? "Create" : "Import"}
                        </Dialog.Close>
                    )}
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    )
}

const CollectionsError: Component = () => {
    const baseClass = "flex items-center p-[1vh] rounded-[1vh] w-full";
    const textClass = "text-sm";
    const containerClass = "md:max-w-1/3 flex flex-col items-center space-y-[2vh]";
    const {status} = useWebSocket();

    return (
        <div class="w-full h-full flex justify-center items-center px-10 md:px-0">
            <div class={containerClass}>
                {((status() === "connecting") || (status() === "reconnecting") ) && (
                    <div class={`${baseClass} border-l-[0.2vw] bg-yellow-600/30 border-yellow-400`}>
                        <div class="pr-[0.75vw] text-yellow-600 w-16 md:w-[3vw]">
                            <InfoSVG />
                        </div>
                        <div>
                            <p class={`${textClass} text-yellow-400`}>Connecting to backend, please wait...</p>
                        </div>
                    </div>
                )}
                {status() === "connected" && (
                    <div class={`${baseClass} border-l-[0.2vw] bg-green-600/30 border-green-400`}>
                        <div class="pr-[0.75vw] text-green-600 w-16 md:w-[3vw]">
                            <InfoSVG />
                        </div>
                        <div>
                            <p class={`${textClass} text-green-400`}>Any collections you create will show up here, click on the 'Create new collection' button to start.</p>
                        </div>
                    </div>
                )}
                {(status() === "error" || status() === "disconnected") && (
                    <div class={`${baseClass} border-1 bg-red-600/30 border-red-400`}>
                        <div class="pr-[0.75vw] text-red-600 w-16 md:w-[3vw]">
                            <ErrorSVG />
                        </div>
                        <div>
                            <p class={`${textClass} text-red-400`}>Failed to connect to server</p>
                        </div>
                    </div>
                )}
            </div>
        </div>
    )
}

const DesktopCollections = () => {
    const ctx = useContext(AppContext)!;
    const { userCollections } = ctx;
    return (
        <DesktopTemplate CurrentPage='Collections'>
            <div class="flex flex-col w-full h-full px-[2vh] p-[1vh] space-y-10">
                <div class="w-full flex justify-between items-center">
                    <p class="text-white font-black text-[4vh]">My Collections</p>
                    <Popup/>
                </div>
                <div class={`w-full ${userCollections().size === 0 ? 'h-full' : 'max-h-full'} flex justify-center flex-wrap space-x-8 space-y-8 overflow-y-auto pt-10 custom-scrollbar`}>
                    <For each={[...userCollections()].sort((a, b) => {
                        const cardA = ctx.knownCollectionCards()[a];
                        const cardB = ctx.knownCollectionCards()[b];
                        if (!cardA || !cardB) return 0;
                        const tsCompare = cardB.timestamp - cardA.timestamp;
                        if (tsCompare !== 0) return tsCompare;
                        return a.localeCompare(b);
                    })} fallback={<CollectionsError />}>
                        {(collection: string) => (
                            <CollectionCard collection={ctx.knownCollectionCards()[collection]} />
                        )}
                    </For>
                </div>
            </div>
        </DesktopTemplate>
    )
}

const MobileCollections = () => {
    const ctx = useContext(AppContext)!;
    const { userCollections } = ctx;
    return (
        <div class="flex flex-col w-full max-h-screen h-screen bg-black">
            <Navbar CurrentPage="Collections" Type="mobile"/>
            <div class="h-[6vh]"/>
            <p class="text-white font-black text-[4vh] px-3">My&nbsp;Collections</p>
            <div class="flex justify-end px-3">
                <Popup/>
            </div>
            <div class="w-full px-4 mt-4 max-h-full h-full flex flex-wrap space-y-4 space-x-4 justify-center overflow-y-auto">
                <For each={[...userCollections()].sort((a, b) => {
                    const cardA = ctx.knownCollectionCards()[a];
                    const cardB = ctx.knownCollectionCards()[b];
                    if (!cardA || !cardB) return 0;
                    const tsCompare = cardB.timestamp - cardA.timestamp;
                    if (tsCompare !== 0) return tsCompare;
                    return a.localeCompare(b);
                })} fallback={<CollectionsError />}>
                    {(collection: string) => (
                        <CollectionCard collection={ctx.knownCollectionCards()[collection]} />
                    )}
                </For>
                <div class="w-full h-[2vh]"/>
            </div>
        </div>
    )
}


const MyCollections = () => {
    const [isMobile, setIsMobile] = createSignal(window.innerWidth <= 768);

    const handleResize = () => {
        setIsMobile(window.innerWidth <= 768);
    };
    window.addEventListener('resize', handleResize);

    return (
        <>
            <title>My Collections | DriveV3</title>
            {isMobile() ? <MobileCollections/> : <DesktopCollections/>}
            <Toaster
            position="bottom-right"
            gutter={8}
            containerClassName=""
            containerStyle={{}}
            toastOptions={{
                className: '',
                duration: 2000,
                style: {
                background: '#363636',
                color: '#fff',
                },
            }}
            />
        </>
    )
}

export default MyCollections