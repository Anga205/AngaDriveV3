import { createSignal } from 'solid-js';
import { DesktopTemplate } from '../components/Template';
import Navbar from '../components/Navbar';
import { BinSVG, CopySVG, EyeSVG } from '../assets/SvgFiles';


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


const DesktopCollections = () => {
    return (
        <DesktopTemplate CurrentPage='Collections'>
            <div class="flex flex-col w-full h-full px-[2vh] p-[1vh] space-y-10">
                <div class="w-full flex justify-between items-center">
                    <p class="text-white font-black text-[4vh]">My Collections</p>
                    <button class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-green-600 hover:bg-green-800 p-[0.2vh] px-[1vh] rounded-[1vh] font-bold translate-y-[4vh]">
                        <span class="text-4xl">+</span>&nbsp;Create Collection
                    </button>
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
        </>
    )
}

export default MyCollections