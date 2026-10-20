import type { Component } from 'solid-js'
import { For } from 'solid-js'
import { useNavigate } from "@solidjs/router";
import { Anga, CollectionSVG, FileSVG, GitHubSVG, HomeSVG, UserSVG } from '../assets/SvgFiles'
import { Pages } from '../library/types';

const DesktopNavbar: Component<{ CurrentPage: Pages }> = (props) => {
    const navigate = useNavigate();

    return (
        <div class="bg-[#161717] h-screen w-25.75 flex flex-col items-center">
            <div class="p-[20%] h-[11vh]" onClick={() => {navigate("/")}}>
                <div class="w-full h-auto">
                    <Anga />
                </div>
            </div>

            <div class="w-full flex flex-col items-end space-y-[0.5vh]">
                <button 
                    class={`w-5/6 ${props.CurrentPage === "Home" ? "bg-black" : "bg-[#161717] hover:bg-black"} pl-4.25 pr-10.25 py-[1.5vh] rounded-l-[1.6vh]`}
                    onClick={() => {navigate("/")}}
                >
                    <HomeSVG />
                </button>
                <button 
                    class={`w-5/6 ${props.CurrentPage === "Files" ? "bg-black" : "bg-[#161717] hover:bg-black"} pl-4.25 pr-10.25 py-[1.5vh] rounded-l-[1.6vh]`}
                    onClick={() => {navigate("/my_drive")}}
                >
                    <FileSVG />
                </button>
                <button 
                    class={`w-5/6 ${props.CurrentPage === "Collections" ? "bg-black" : "bg-[#161717] hover:bg-black"} pl-4.25 pr-10.25 py-[1.5vh] rounded-l-[1.6vh]`}
                    onClick={() => {navigate("/my_collections")}}
                >
                    <CollectionSVG />
                </button>
                <a 
                    href="https://github.com/Anga205/AngaDriveV3"
                    target="_blank"
                    class="w-5/6 bg-[#161717] hover:bg-black pl-4.25 pr-10.25 py-[1.5vh] rounded-l-[1.6vh]"
                >
                    <GitHubSVG />
                </a>
            </div>
            <div class="grow" />
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

    const items: Array<{ page: Pages; label: string; route: string; icon: Component }> = [
        { page: "Home", label: "Home", route: "/", icon: HomeSVG },
        { page: "Files", label: "Files", route: "/my_drive", icon: FileSVG },
        { page: "Collections", label: "Collections", route: "/my_collections", icon: CollectionSVG },
        { page: "Account", label: "Account", route: "/account", icon: UserSVG },
    ];

    return (
        <nav
            aria-label="Main"
            class="fixed bottom-0 left-0 right-0 z-30 h-16 bg-[#161717] flex items-stretch"
        >
            <For each={items}>
                {(item) => {
                    const isActive = () => props.CurrentPage === item.page;
                    return (
                        <button
                            type="button"
                            aria-label={item.label}
                            aria-current={isActive() ? "page" : undefined}
                            class={`flex-1 flex flex-col items-center justify-center gap-1 border-t-2 transition-colors ${
                                isActive()
                                    ? "border-blue-500 bg-black text-white"
                                    : "border-[#242424] text-neutral-500 hover:text-white"
                            }`}
                            onClick={() => { navigate(item.route); }}
                        >
                            <div class={`w-6 h-6 ${isActive() ? "opacity-100" : "opacity-60"}`}>
                                <item.icon />
                            </div>
                            <span class="text-[11px] font-bold leading-none">{item.label}</span>
                        </button>
                    );
                }}
            </For>
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