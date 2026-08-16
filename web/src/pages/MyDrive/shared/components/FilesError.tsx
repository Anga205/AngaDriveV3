import { Component } from "solid-js";
import { useWebSocket } from "@/Websockets";
import { ErrorSVG, InfoSVG } from "@/assets/SvgFiles";


const FilesError: Component = () => {
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
                    <div class={`${baseClass} border-l-[0.2vw] bg-blue-600/30 border-blue-400`}>
                        <div class="pr-[0.75vw] text-blue-600 w-16 md:w-[3vw]">
                            <InfoSVG />
                        </div>
                        <div>
                            <p class={`${textClass} text-blue-400`}>Any files you upload will show up here, click on the &apos;Upload&apos; button to start uploading files or Drag & Drop files anywhere on this website</p>
                        </div>
                    </div>
                )}
                {(status() === "error" || status() === "disconnected") && (
                    <div class={`${baseClass} border bg-red-600/30 border-red-400`}>
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

export default FilesError;