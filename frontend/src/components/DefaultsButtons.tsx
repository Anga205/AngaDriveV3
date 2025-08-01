import type { Component } from "solid-js";
import { createSignal } from "solid-js";
import { DatabaseZapSVG, LockSVG, ScanEyeSVG } from "../assets/SvgFiles";


const DefaultButton: Component<{ bgColor: string, title: string, status: boolean, Icon: Component }> = (props) => {

    const [enabled, setEnabled]: [() => boolean, (value: boolean) => void] = createSignal(props.status);

    return (
        <div 
            class="w-1/3 bg-[#242424] h-[15vh] rounded-[1.5vh] overflow-hidden transform transition-transform duration-300 active:scale-95 hover:brightness-150" 
            style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);"
            onClick={() => {setEnabled(!enabled())}}
            onContextMenu={(e: MouseEvent) => {
                e.preventDefault();
                setEnabled(props.status);
            }}
        >
            <div class={`w-full h-[0.5vh] ${props.bgColor}`}/>
            <div class="flex p-[1.3vh] px-[2vh] w-full h-full">
                <div class="h-full flex flex-col">
                    <p class="text-white font-semibold text-[1vw]">{props.title}</p>
                    <div class="flex-grow"/>
                    <p class="text-gray-400 text-[0.8vw] mb-[1vh]">Default: {enabled() ? "Enabled" : "Disabled"}</p>
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
            <DefaultButton bgColor="bg-blue-500" title="Caching" status={false} Icon={DatabaseZapSVG} />
            <DefaultButton bgColor="bg-purple-700" title="File Previews" status={true} Icon={ScanEyeSVG} />
            <DefaultButton bgColor="bg-green-600" title="Ultra Secure" status={false} Icon={LockSVG} />
        </div>
    )
}

const MobileDefaults: Component<{ bgColor: string, title: string, status: boolean, Icon: Component}> = (props) => {
    const [enabled, setEnabled]: [() => boolean, (value: boolean) => void] = createSignal(props.status);
    const [isClicked, setIsClicked] = createSignal(false);

    const handleClick = () => {
        setIsClicked(true);
        setTimeout(() => setIsClicked(false), 200);
        setEnabled(!enabled());
    };

    return (
        <div 
            class={`flex flex-col w-1/3 rounded-xl aspect-square bg-[#242424] overflow-hidden transform transition-transform duration-200 ${isClicked() ? "scale-95" : "scale-100"}`}
            onClick={handleClick}
            onContextMenu={(e: MouseEvent) => {
                e.preventDefault();
                setEnabled(props.status);
            }}
        >
            <div class={`w-full h-1 ${props.bgColor}`}/>
            <div class="w-full h-full p-1 flex flex-col space-y-[1vh] justify-center items-center">
                <p class="font-black text-white text-[3vw]">{props.title}</p>
                <div class="flex-grow aspect-square">
                    <props.Icon />
                </div>
                <p class="text-gray-400 text-[2vw]">Default: {enabled() ? "Enabled" : "Disabled"}</p>
            </div>
        </div>
    )
}

const MobileButtons: Component = () => {
    return (
        <div class="w-full flex space-x-[1.5vh] px-[1.5vh] opacity-95">
            <MobileDefaults bgColor="bg-blue-500" title="Caching" status={false} Icon={DatabaseZapSVG} />
            <MobileDefaults bgColor="bg-purple-700" title="File Previews" status={true} Icon={ScanEyeSVG} />
            <MobileDefaults bgColor="bg-green-600" title="Ultra Secure" status={false} Icon={LockSVG} />
        </div>
    )
}

export {DefaultsButtons, MobileButtons}