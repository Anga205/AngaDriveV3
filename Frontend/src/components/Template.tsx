import { Component } from "solid-js";
import Navbar from "./Navbar";

const DesktopTemplate: Component<{ children: any }> = (props) => {
    return (
        <div class="max-h-screen w-screen flex items-start bg-black overflow-hidden">
            <Navbar CurrentPage="Files"/>
            <div class="flex flex-col w-[95.5vw] h-screen pl-[2vh] pr-[1vh] py-[1vh] space-y-[1.5vh]">
                {props.children}
            </div>
        </div>
    );
}

export {DesktopTemplate}