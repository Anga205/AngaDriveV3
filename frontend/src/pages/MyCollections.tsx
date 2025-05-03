import { createSignal } from 'solid-js';
import { DesktopTemplate } from '../components/Template';
import Navbar from '../components/Navbar';

const DesktopCollections = () => {
    return (
        <DesktopTemplate CurrentPage='Collections'>
            <div class="flex w-full h-full pl-[2vh] pr-[1vh] py-[1vh]"/>
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