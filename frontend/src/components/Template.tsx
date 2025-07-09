import type { Component } from "solid-js";
import Navbar from "./Navbar";
import type { Pages } from "../library/types";

const DesktopTemplate: Component<{ CurrentPage: Pages; children: any }> = (props) => {
    return (
        <div class="max-h-screen w-screen h-screen flex items-start bg-black overflow-hidden">
            <Navbar CurrentPage={props.CurrentPage} Type="desktop"/>
            <div class="w-[calc(100vw-103px)] h-full">
                {props.children}
            </div>
        </div>
    );
}

export {DesktopTemplate}