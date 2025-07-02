import { createEffect, createSignal } from 'solid-js';
import { DesktopTemplate } from '../components/Template';
import Navbar from '../components/Navbar';
import { BinSVG, CopySVG, EyeSVG } from '../assets/SvgFiles';
import Dialog from '@corvu/dialog';
import { useWebSocket } from '../Websockets';
import toast, { Toaster } from 'solid-toast';


const CollectionCard = () => {
    return (
        <div class="flex flex-col w-64 h-60 bg-neutral-800 rounded-lg px-4 py-2 pb-3">
            <p class="text-white font-bold text-2xl text-center w-full p-2">Collection Name</p>
            <hr class="border-neutral-600"/>
            <div class="flex w-full items-center justify-center h-full my-2">
                <div class="flex flex-col items-start justify-between h-full">
                    <p class="text-gray-400 text-sm">Size:</p>
                    <p class="text-gray-400 text-sm">File Count:</p>
                    <p class="text-gray-400 text-sm">Folder Count:</p>
                    <p class="text-gray-400 text-sm">Editors:</p>
                </div>
                <div class="flex flex-col items-start justify-between h-full ml-4">
                    <p class="text-white text-sm">10 GB</p>
                    <p class="text-white text-sm">100</p>
                    <p class="text-white text-sm">5</p>
                    <p class="text-white text-sm text-start">4</p>
                </div>
            </div>
            <div class="flex w-full justify-between px-5">
                <a class="flex items-center justify-center p-2 bg-yellow-700/30 hover:bg-yellow-700/20 rounded-xl text-yellow-600">
                    <EyeSVG />
                </a>
                <a class="flex items-center justify-center p-2 bg-lime-700/30 hover:bg-lime-700/20 rounded-xl text-lime-600">
                    <CopySVG />
                </a>
                <a class="flex items-center justify-center p-2 bg-red-700/30 hover:bg-red-700/20 rounded-xl text-red-600">
                    <BinSVG />
                </a>
            </div>
        </div>
    )
}

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
            socket.send(JSON.stringify({
                type: "new_collection",
                data: {
                    collection_name: newCollectionName(),
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
            <Dialog.Trigger class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-green-600 hover:bg-green-800 p-[0.2vh] px-[1vh] rounded-[1vh] font-bold translate-y-[4vh]">
                <span class="text-4xl text-center">+</span>&nbsp;Create new collection
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
                        <button onClick={onSubmit} class="bg-green-700 text-white p-2 rounded-lg font-semibold w-full hover:bg-green-800 transition-colors disabled:bg-gray-500 disabled:cursor-not-allowed">
                            Submit
                        </button>
                    )}
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    )
}

const DesktopCollections = () => {
    return (
        <DesktopTemplate CurrentPage='Collections'>
            <div class="flex flex-col w-full h-full px-[2vh] p-[1vh] space-y-10">
                <div class="w-full flex justify-between items-center">
                    <p class="text-white font-black text-[4vh]">My Collections</p>
                    <Popup/>
                </div>
                <div class="flex flex-wrap space-x-4 space-y-4 justify-center pt-3 overflow-y-scroll">
                    <CollectionCard/>
                    <CollectionCard/>
                    <CollectionCard/>
                    <CollectionCard/>
                    <CollectionCard/>
                    <CollectionCard/>
                    <CollectionCard/>
                </div>
            </div>
        </DesktopTemplate>
    )
}

const MobileCollections = () => {
    return (
        <div class="flex flex-col w-full min-h-screen h-screen bg-black">
            <Navbar CurrentPage="Collections" Type="mobile"/>
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
            <title>My Files | DriveV3</title>
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