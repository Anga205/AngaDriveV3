import type { Component } from "solid-js"
import { createSignal } from "solid-js"
import { DesktopTemplate } from "../components/Template"
import { InfoSVG, UploadSVG, ErrorSVG } from "../assets/SvgFiles"
import Navbar from "../components/Navbar";

interface FilesNotFoundProps {
    status: "noFiles" | "error";
}

const FilesError: Component<FilesNotFoundProps> = (props) => {
    const baseClass = "flex items-center p-[1vh] rounded-[1vh] w-full";
    const textClass = "text-[1.5vh]";
    const containerClass = "md:max-w-1/3 flex flex-col items-center space-y-[2vh]";

    return (
        <div class={containerClass}>
            {props.status === "noFiles" && (
                <div class={`${baseClass} border-l-4 bg-blue-600/30 border-blue-400`}>
                    <div class="pr-[1.5vh] text-blue-600">
                        <InfoSVG />
                    </div>
                    <div>
                        <p class={`${textClass} text-blue-400`}>Any files you upload will show up here, click on the &apos;Upload&apos; button to start uploading files or Drag & Drop files anywhere on this website</p>
                    </div>
                </div>
            )}
            {props.status === "error" && (
                <div class={`${baseClass} border-1 bg-red-600/30 border-red-400`}>
                    <div class="pr-[1.5vh] text-red-600">
                        <ErrorSVG />
                    </div>
                    <div>
                        <p class={`${textClass} text-red-400`}>Failed to connect to backend</p>
                    </div>
                </div>
            )}
        </div>
    )
}



const DesktopDrive: Component = () => {
    return (
        <DesktopTemplate CurrentPage="Files">
            <div class="flex flex-col w-full h-full px-[2vh] p-[1vh]">
                <div class="w-full flex justify-between items-center">
                    <p class="text-white font-black text-[4vh]">My Files</p>
                    <button class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-blue-600 hover:bg-blue-800 p-[1vh] rounded-[1vh] font-bold translate-y-[4vh]"><UploadSVG/>Upload</button>
                </div>
                <div class="w-full h-full flex justify-center items-center">
                    <FilesError status="error"/>
                </div>
            </div>
        </DesktopTemplate>
    )
}

const MobileDrive: Component = () => {
    return (
        <div class="flex flex-col w-full min-h-screen h-screen bg-black">
            <Navbar CurrentPage="Files" Type="mobile"/>
            <div class="h-[5vh]"/>
            <div class="w-full flex justify-between items-center px-[3vw]">
                <p class="text-white font-black text-[4vh]">My Files</p>
                <button class="cursor-pointer hover:text-gray-300 text-white flex justify-center items-center bg-blue-600 hover:bg-blue-800 p-[1vh] rounded-[1vh] font-bold translate-y-[4vh]"><UploadSVG/>Upload</button>
            </div>
            <div class="w-full h-full flex justify-center items-center">
                <FilesError status="error"/>
            </div>
        </div>
    )
}

const MyDrive: Component = () => {
    const [isMobile, setIsMobile] = createSignal(window.innerWidth <= 768);

    const handleResize = () => {
        setIsMobile(window.innerWidth <= 768);
    };
    window.addEventListener('resize', handleResize);
    return (
        <>
            <title>My Files | DriveV3</title>
            {isMobile() ? <MobileDrive /> : <DesktopDrive />}
        </>
    )
}

export default MyDrive