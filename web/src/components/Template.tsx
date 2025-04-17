import type { Component } from "solid-js";
import Navbar from "./Navbar";

const DesktopTemplate: Component<{ CurrentPage: string; children: any }> = (props) => {
    return (
        <div class="max-h-screen w-screen h-screen flex items-start bg-black overflow-hidden">
            <Navbar CurrentPage={props.CurrentPage} Type="desktop"/>
            {props.children}
        </div>
    );
}

export {DesktopTemplate}