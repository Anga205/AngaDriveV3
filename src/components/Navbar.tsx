import type { Component } from 'solid-js'
import { CollectionSVG, FileSVG, GitHubSVG, HomeSVG, UserSVG } from '../assets/SvgFiles'

const Navbar: Component = () => {
    return (
        <div class="bg-[#161717] h-screen w-[4.5vw] border-white flex flex-col items-center">
            <div class="p-[20%] h-[10vh]">
                <img src="/anga.svg" alt="Anga Logo" class="w-full h-auto" />
            </div>

            <div class="flex w-full mt-[10%] justify-end">
                <div class="flex w-5/6 bg-black hover:bg-[#161717] rounded-l-2xl">
                    <div class="p-[20%] h-full aspect-square">
                        <HomeSVG />
                    </div>
                </div>
            </div>
            <div class="flex w-full mt-[5%] justify-end">
                <div class="flex w-5/6 bg-[#161717] hover:bg-black rounded-l-2xl">
                    <div class="p-[20%]">
                        <FileSVG />
                    </div>
                </div>
            </div>
            <div class="flex w-full mt-[5%] justify-end">
                <div class="flex w-5/6 bg-[#161717] hover:bg-black rounded-l-2xl">
                    <div class="p-[20%]">
                        <CollectionSVG />
                    </div>
                </div>
            </div>
            <div class="flex w-full mt-[5%] justify-end">
                <div class="flex w-5/6 bg-[#161717] hover:bg-black rounded-l-2xl">
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