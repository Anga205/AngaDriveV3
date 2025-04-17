import type { Component } from "solid-js";
import Navbar from "./Navbar";

const DesktopTemplate: Component<{ children: any }> = (props) => {
    return (
        <div class="max-h-screen w-screen h-screen flex items-start bg-black overflow-hidden">
            <Navbar CurrentPage="Files"/>
            {props.children}
        </div>
    );
}

export {DesktopTemplate}