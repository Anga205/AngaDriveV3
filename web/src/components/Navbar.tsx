import type { Component } from 'solid-js'
import { For } from 'solid-js'
import { useLocation, useNavigate } from "@solidjs/router";
import { Anga, CollectionSVG, FileSVG, GitHubSVG, HomeSVG, UserSVG } from '../assets/SvgFiles'
import { MOBILE_TABS, pageFromPath, pagerPosition } from '../library/tabs';
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


const MobileNavbar: Component = () => {
    const navigate = useNavigate();
    const location = useLocation();

    // Settled page, derived from the URL. Used for accessibility only — the
    // visual highlight follows the live swipe position instead.
    const currentPage = () => pageFromPath(location.pathname);

    const count = MOBILE_TABS.length;

    // Live fractional position, clamped to the valid pane range.
    const position = () => Math.max(0, Math.min(count - 1, pagerPosition()));

    // 1 when the pager sits exactly on this tab, 0 when it is a full pane away.
    // Halfway between two tabs gives 0.5 for both.
    const intensity = (index: number) => Math.max(0, 1 - Math.abs(position() - index));

    return (
        <nav
            aria-label="Main"
            class="fixed bottom-0 left-0 right-0 z-30 h-16 bg-[#161717] flex items-stretch"
        >
            {/* Sliding highlight bar. Width is one tab; translateX moves it by
                its own width, so `position * 100%` lands it exactly between
                tabs mid-swipe. No CSS transition — scroll events drive it. */}
            <div
                class="pointer-events-none absolute top-0 left-0 z-10 h-0.5 bg-blue-500"
                style={{
                    width: `${100 / count}%`,
                    transform: `translateX(${position() * 100}%)`,
                }}
            />
            <For each={MOBILE_TABS}>
                {(item, index) => {
                    const isActive = () => currentPage() === item.page;
                    const level = () => intensity(index());
                    return (
                        <button
                            type="button"
                            aria-label={item.label}
                            aria-current={isActive() ? "page" : undefined}
                            class="relative flex-1 flex flex-col items-center justify-center gap-1 border-t-2 border-[#242424] text-white"
                            onClick={() => { navigate(item.route, { scroll: false }); }}
                        >
                            {/* Background darkens as the tab becomes active. */}
                            <div
                                class="pointer-events-none absolute inset-0 bg-black"
                                style={{ opacity: level() }}
                            />
                            <div
                                class="relative w-6 h-6"
                                style={{ opacity: 0.5 + 0.5 * level() }}
                            >
                                <item.icon />
                            </div>
                            <span
                                class="relative text-[11px] font-bold leading-none"
                                style={{ opacity: 0.5 + 0.5 * level() }}
                            >
                                {item.label}
                            </span>
                        </button>
                    );
                }}
            </For>
        </nav>
    );
};

const Navbar: Component<{ CurrentPage?: Pages, Type: string }> = (props) => {
    return (
        <>
            {
                props.Type === "desktop" ? (
                    <DesktopNavbar CurrentPage={props.CurrentPage ?? "Home"} />
                ) : (
                    <MobileNavbar />
                )
            }
        </>
    )
}

export default Navbar