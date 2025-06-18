import type { Component } from "solid-js";
import Navbar from "./Navbar";
import type { Pages } from "../library/types";

const DesktopTemplate: Component<{ CurrentPage: Pages; children: any }> = (props) => {
    return (
        <div class="max-h-screen w-screen h-screen flex items-start bg-black overflow-hidden">
            <Navbar CurrentPage={props.CurrentPage} Type="desktop"/>
            {props.children}
        </div>
    );
}

export {DesktopTemplate}