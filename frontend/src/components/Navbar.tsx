import type { Component } from 'solid-js'
import { useNavigate } from "@solidjs/router";
import { Anga, CollectionSVG, FileSVG, GitHubSVG, HomeSVG, UserSVG, HamburgerSVG } from '../assets/SvgFiles'
import Drawer from '@corvu/drawer'
import { Pages } from '../types/types';

const DesktopNavbar: Component<{ CurrentPage: Pages }> = (props) => {
    const navigate = useNavigate();

    return (
        <div class="bg-[#161717] h-screen w-[4.5vw] flex flex-col items-center">
            <div class="p-[20%] h-[11vh]" onClick={() => {navigate("/")}}>
                <div class="w-full h-auto">
                    <Anga />
                </div>
            </div>

            <div class="w-full flex flex-col items-end space-y-[0.5vh]">
                <button 
                    class={`w-5/6 ${props.CurrentPage === "Home" ? "bg-black" : "bg-[#161717] hover:bg-black"} pl-[0.7vw] pr-[1.7vw] py-[1.5vh] rounded-l-[1.6vh]`}
                    onClick={() => {navigate("/")}}
                >
                    <HomeSVG />
                </button>
                <button 
                    class={`w-5/6 ${props.CurrentPage === "Files" ? "bg-black" : "bg-[#161717] hover:bg-black"} pl-[0.7vw] pr-[1.7vw] py-[1.5vh] rounded-l-[1.6vh]`}
                    onClick={() => {navigate("/my_drive")}}
                >
                    <FileSVG />
                </button>
                <button 
                    class={`w-5/6 ${props.CurrentPage === "Collections" ? "bg-black" : "bg-[#161717] hover:bg-black"} pl-[0.7vw] pr-[1.7vw] py-[1.5vh] rounded-l-[1.6vh]`}
                    onClick={() => {navigate("/my_collections")}}
                >
                    <CollectionSVG />
                </button>
                <a 
                    href="https://github.com/Anga205/AngaDriveV3"
                    target="_blank"
                    class="w-5/6 bg-[#161717] hover:bg-black pl-[0.7vw] pr-[1.7vw] py-[1.5vh] rounded-l-[1.6vh]"
                >
                    <GitHubSVG />
                </a>
            </div>
            <div class="flex-grow" />
            <div class="w-full p-[16%]" onClick={() => {navigate("/account")}}>
                <div class="hover:bg-black p-[20%] rounded-full">
                    <UserSVG />
                </div>
            </div>
        </div>
    )
}


const MobileNavbar: Component<{ CurrentPage: Pages }> = (props) => {
    const navigate = useNavigate();
    return (
        <nav class="backdrop-blur-md w-full h-[5vh] border-b-2 border-[#242424] flex items-center fixed z-10">
            <Drawer snapPoints={[0, 1]} allowSkippingSnapPoints={false} side='left'>
                {(drawerProps) => (
                    <>
                        <Drawer.Trigger class="h-full aspect-square flex justify-center items-center p-[1vh]">
                            <HamburgerSVG />
                        </Drawer.Trigger>
                        <Drawer.Portal>
                            <Drawer.Overlay
                                class="bg-black fixed inset-0 z-50 data-transitioning:transition-colors data-transitioning:duration-500 data-transitioning:ease-[cubic-bezier(0.32,0.72,0,1)]"
                                style={{
                                    'background-color': `rgb(0 0 0 / ${
                                        0.5 * drawerProps.openPercentage
                                    })`,
                                }}
                            />
                            <Drawer.Content class="bg-[#161717] fixed inset-y-0 left-0 z-50 flex w-full max-w-[75vw] flex-col rounded-r-lg border-r-4 border-[#242424] pt-3 after:absolute after:inset-y-0 after:right-[calc(100%-1px)] after:w-1/2 after:bg-inherit data-transitioning:transition-transform data-transitioning:duration-500 data-transitioning:ease-[cubic-bezier(0.32,0.72,0,1)] md:select-none">
                                <div class="h-1/6 flex justify-center items-center space-x-[3vh] select-none">
                                    <div class="h-1/2 aspect-square">
                                        <Anga />
                                    </div>
                                    <p class="font-black text-white text-4xl">DriveV3</p>
                                </div>
                                <div class="mt-4 flex flex-col items-start space-y-4 px-4">
                                    <button
                                        class={`w-full text-left px-4 py-2 flex items-center space-x-4 rounded-lg ${
                                            props.CurrentPage === "Home" ? "bg-black text-white" : "hover:bg-[#242424] text-white"
                                        }`}
                                        onClick={() => {
                                            navigate("/");
                                        }}
                                    >
                                        <div class="h-full aspect-square">
                                            <HomeSVG />
                                        </div>
                                        <span class="font-bold">Home</span>
                                    </button>
                                    <button
                                        class={`w-full text-left px-4 py-2 flex items-center space-x-4 rounded-lg ${
                                            props.CurrentPage === "Files" ? "bg-black text-white" : "hover:bg-[#242424] text-white"
                                        }`}
                                        onClick={() => {
                                            navigate("/my_drive");
                                        }}
                                    >
                                        <div class="h-full aspect-square">
                                            <FileSVG />
                                        </div>
                                        <span class="font-bold">Files</span>
                                    </button>
                                    <button
                                        class={`w-full text-left px-4 py-2 flex items-center space-x-4 rounded-lg ${
                                            props.CurrentPage === "Collections" ? "bg-black text-white" : "hover:bg-[#242424] text-white"
                                        }`}
                                        onClick={() => {
                                            navigate("/my_collections");
                                        }}
                                    >
                                        <div class="h-full aspect-square">
                                            <CollectionSVG />
                                        </div>
                                        <span class="font-bold">Collections</span>
                                    </button>
                                    <button
                                        class="w-full text-left px-4 py-2 flex items-center space-x-4 rounded-lg text-white hover:bg-[#242424]"
                                        onClick={() => {
                                            window.open("https://github.com/Anga205/AngaDriveV3", "_blank");
                                        }}
                                    >
                                        <div class="h-full aspect-square">
                                            <GitHubSVG />
                                        </div>
                                        <span class="font-bold">GitHub</span>
                                    </button>
                                </div>
                            </Drawer.Content>
                        </Drawer.Portal>
                    </>
                )}
            </Drawer>
        </nav>
    );
};

const Navbar: Component<{ CurrentPage: Pages, Type: string }> = (props) => {
    return (
        <>
            {
                props.Type === "desktop" ? (
                    <DesktopNavbar CurrentPage={props.CurrentPage} />
                ) : (
                    <MobileNavbar CurrentPage={props.CurrentPage} />
                )
            }
        </>
    )
}

export default Navbar