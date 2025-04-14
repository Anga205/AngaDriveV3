import type { Component } from 'solid-js'
import { useNavigate } from "@solidjs/router";
import { CollectionSVG, FileSVG, GitHubSVG, HomeSVG, UserSVG } from '../assets/SvgFiles'

const Navbar: Component<{ CurrentPage?: string }> = (props) => {
    const currentPage = props.CurrentPage || "Home";
    const navigate = useNavigate();

    return (
        <div class="bg-[#161717] h-screen w-[4.5vw] border-white flex flex-col items-center">
            <div class="p-[20%] h-[10vh]">
                <img src="/anga.svg" alt="Anga Logo" class="w-full h-auto" />
            </div>

            <div class="flex w-full mt-[10%] justify-end">
                <div 
                    class={`flex w-5/6 ${currentPage === "Home" ? "bg-black" : "bg-[#161717] hover:bg-black"} rounded-l-2xl`}
                    onClick={() => {
                        navigate("/")
                    }}>
                    <div class="p-[20%] h-full aspect-square">
                        <HomeSVG />
                    </div>
                </div>
            </div>
            <div class="flex w-full mt-[5%] justify-end">
                <div 
                    class={`flex w-5/6 ${currentPage === "Files" ? "bg-black" : "bg-[#161717] hover:bg-black"} rounded-l-2xl`}
                    onClick={() => {
                        navigate("/my_drive")
                    }}>
                    <div class="p-[20%]">
                        <FileSVG />
                    </div>
                </div>
            </div>
            <div class="flex w-full mt-[5%] justify-end">
                <div class={`flex w-5/6 ${currentPage === "Collections" ? "bg-black" : "bg-[#161717] hover:bg-black"} rounded-l-2xl`}>
                    <div class="p-[20%]">
                        <CollectionSVG />
                    </div>
                </div>
            </div>
            <div class="flex w-full mt-[5%] justify-end">
                <div 
                    class={`flex w-5/6 ${currentPage === "GitHub" ? "bg-black" : "bg-[#161717] hover:bg-black"} rounded-l-2xl`}
                    onClick={() => {
                        window.open("https://github.com/Anga205/AngaDriveV3", "_blank");
                    }}>
                    <div class="p-[20%]">
                        <GitHubSVG />
                    </div>
                </div>
            </div>
            <div class="flex-grow" />
            <div class="w-full p-[32%]">
                <UserSVG />
            </div>
        </div>
    )
}

export default Navbar