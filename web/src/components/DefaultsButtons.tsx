import type { Component } from "solid-js";
import { DatabaseZapSVG, LockSVG, ScanEyeSVG } from "../assets/SvgFiles";


const DefaultButton: Component<{ bgColor: string, title: string, status: string, Icon: Component }> = (props) => {
    return (
        <div class="w-1/3 bg-[#242424] h-[15vh] rounded-[1.5vh] overflow-hidden" style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);">
            <div class={`w-full h-[0.5vh] ${props.bgColor}`}/>
            <div class="flex p-[1.3vh] px-[2vh] w-full h-full">
                <div class="h-full flex flex-col">
                    <p class="text-white font-semibold text-[1vw]">{props.title}</p>
                    <div class="flex-grow"/>
                    <p class="text-gray-400 text-[0.8vw] mb-[1vh]">Default: {props.status}</p>
                </div>
                <div class="flex-grow"/>
                <div class="h-full aspect-square py-[2vh] pl-[2.5vw]">
                    <props.Icon />
                </div>
            </div>
        </div>
    )
}

const DefaultsButtons: Component = () => {
    return (
        <div class="w-full flex space-x-[1.5vh] h-[15vh]">
            <DefaultButton bgColor="bg-blue-500" title="Caching" status="Disabled" Icon={DatabaseZapSVG} />
            <DefaultButton bgColor="bg-purple-700" title="File Previews" status="Enabled" Icon={ScanEyeSVG} />
            <DefaultButton bgColor="bg-green-600" title="Ultra Secure" status="Disabled" Icon={LockSVG} />
        </div>
    )
}

const MobileDefaults: Component<{ bgColor: string, title: string, status: string, Icon: Component}> = (props) => {
    return (
        <div class="flex flex-col w-1/3 rounded-xl aspect-square bg-[#242424] overflow-hidden">
            <div class={`w-full h-1 ${props.bgColor}`}/>
            <div class="w-full h-full p-1 flex flex-col space-y-[1vh] justify-center items-center">
                <p class="font-black text-white text-[3vw]">{props.title}</p>
                <div class="flex-grow aspect-square">
                    <props.Icon />
                </div>
                <p class="text-gray-400 text-[2vw]">Default: {props.status}</p>
            </div>
        </div>
    )
}

const MobileButtons: Component = () => {
    return (
        <div class="w-full flex space-x-[1.5vh] px-[1.5vh] opacity-95">
            <MobileDefaults bgColor="bg-blue-500" title="Caching" status="Disabled" Icon={DatabaseZapSVG} />
            <MobileDefaults bgColor="bg-purple-700" title="File Previews" status="Enabled" Icon={ScanEyeSVG} />
            <MobileDefaults bgColor="bg-green-600" title="Ultra Secure" status="Disabled" Icon={LockSVG} />
        </div>
    )
}

export {DefaultsButtons, MobileButtons}