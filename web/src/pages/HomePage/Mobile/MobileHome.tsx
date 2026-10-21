import { Component, Accessor } from "solid-js";
import { MobileButtons } from "@/components/DefaultsButtons";
import { MobileHeader } from "@/components/Header";
import { RAMUsage, CPUUsage } from "@/components/CircularProgress";
import GraphComponent from "@/components/GraphComponent";
import { Butterfly } from "@/assets/SvgFiles";
import ContactMe from "@/components/ContactMe";
import type { CPUData, GraphData, RAMData } from "@/library/types";
import { UserCount, FilesHosted } from "../shared/Stats";

const MobileHome: Component<{ ramdata: RAMData; cpudata: CPUData; siteActivity: Accessor<GraphData>; spaceUsed: Accessor<GraphData>; userCount: Accessor<number>; filesHosted: Accessor<number> }> = (props) => {
    return (
        <div class="relative w-full h-full bg-black overflow-hidden">
            {/* Non-scrolling background layer */}
            <div class="absolute inset-0 flex items-center justify-center p-[5%] pointer-events-none">
                <div class="w-full opacity-30">
                    <Butterfly/>
                </div>
            </div>
            {/* Scrolling content layer. Only vertical overscroll is contained so
                that horizontal swipes still chain to the tab pager. */}
            <div class="relative w-full h-full overflow-y-auto overscroll-y-contain">
                <div class="w-full space-y-[1vh] pt-4 pb-16">
                    <MobileHeader />
                    <MobileButtons />
                    <div class="w-full px-[1.5vh]">
                        <div class="w-full bg-[#242424] aspect-video rounded-xl p-[1vw] opacity-95">
                            <p class="text-center font-black text-white text-[4vw]">Site Activity Over Past Month</p>
                            <GraphComponent GraphData={props.siteActivity}/>
                        </div>
                    </div>
                    <div class="w-full px-[1.5vh] space-x-[1.5vh] opacity-95 flex">
                        <div class="flex flex-col justify-center items-center w-1/2 aspect-square bg-[#242424] rounded-xl">
                            <p class="text-center w-full font-black text-white">
                                RAM Usage
                            </p>
                            <RAMUsage data={props.ramdata} />
                        </div>
                        <div class="flex flex-col justify-center items-center w-1/2 aspect-square bg-[#242424] rounded-xl">
                            <p class="text-center w-full font-black text-white">
                                CPU Usage
                            </p>
                            <CPUUsage data={props.cpudata}/>
                        </div>
                    </div>
                    <div class="w-full px-[1.5vh] opacity-95">
                        <div class="w-full aspect-64/27 p-[1vw] bg-[#242424] rounded-xl">
                            <p class="w-full font-black text-white text-center">
                                Space Used
                            </p>
                            <GraphComponent GraphData={props.spaceUsed}/>
                        </div>
                    </div>
                    <div class="w-full px-[1.5vh] opacity-95 flex space-x-[1.5vh] pb-[1.5vh]">
                        <div class="w-1/3 flex flex-col space-y-[1.5vh]">
                            <UserCount isMobile={true} count={props.userCount} />
                            <FilesHosted isMobile={true} count={props.filesHosted} />
                        </div>
                        <ContactMe />
                    </div>
                </div>
            </div>
        </div>
    )
}

export default MobileHome;