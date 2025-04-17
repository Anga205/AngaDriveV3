import type { Component } from "solid-js"
import { DesktopTemplate } from "../components/Template"
import { UploadSVG } from "../assets/SvgFiles"

const DesktopDrive: Component = () => {
    return (
        <DesktopTemplate>
            <div class="w-full h-full px-[2vh] p-[1vh]">
                <div class="w-full flex justify-between">
                    <p class="text-white font-black text-[4vh]">My Drive</p>
                    <button class="h-full text-white flex bg-blue-600 p-[1vh] rounded-[1vh] font-bold translate-y-[4vh]"><UploadSVG/>Upload</button>
                </div>
            </div>
        </DesktopTemplate>
    )
}

export {DesktopDrive}