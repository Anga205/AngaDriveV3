import { Component, Accessor } from "solid-js";
import { DesktopTemplate } from "@/components/Template";
import { DefaultsButtons } from "@/components/DefaultsButtons";
import { Header } from "@/components/Header";
import { RAMUsage, CPUUsage } from "@/components/CircularProgress";
import GraphComponent from "@/components/GraphComponent";
import { Butterfly } from "@/assets/SvgFiles";
import ContactMe from "@/components/ContactMe";
import type { CPUData, GraphData, RAMData } from "@/library/types";
import { UserCount, FilesHosted } from "../shared/Stats";

const DesktopHome: Component<{ ramdata: RAMData; cpudata: CPUData; siteActivity: Accessor<GraphData>; spaceUsed: Accessor<GraphData>; userCount: Accessor<number>; filesHosted: Accessor<number>; }> = (props) => {
    return (
        <DesktopTemplate CurrentPage="Home">
            <div class="flex flex-col w-full h-full pl-[2vh] pr-[1vh] py-[1vh] space-y-[1.5vh]">
                <Header />
                <div class="flex w-full h-full space-x-[1.5vh]">
                    <div class="flex flex-col w-1/2 h-full space-y-[1.5vh]">
                        <DefaultsButtons />
                        <div class="h-[71.5vh] space-y-[1.5vh] pb-[1.5vh]">
                            <div class="w-full h-5/12 max-h-5/12 bg-[#242424] flex flex-col rounded-[1.5vh] pt-[2.5vh] p-[1.65vh] overflow-hidden justify-center items-center" style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);">
                                <p class="text-white font-semibold text-[2vh]">Site Activity Over Past Month</p>
                                <GraphComponent GraphData={props.siteActivity}/>
                            </div>
                            <div class="w-full h-7/12 flex space-x-[1.5vh]">
                                <div class="w-1/2 bg-[#242424] rounded-[1.5vh] p-[1.65vh]" style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);">
                                    <p class="text-white font-semibold text-[2vh]">Ram Usage</p>
                                    <div class="w-full h-full flex justify-center items-center overflow-hidden">
                                        <RAMUsage data={props.ramdata} />
                                    </div>
                                </div>
                                <div class="w-1/2 bg-[#242424] rounded-[1.5vh] p-[1.65vh]" style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);">
                                    <p class="text-white font-semibold text-[2vh]">CPU Usage</p>
                                    <div class="w-full h-full flex justify-center items-center overflow-hidden">
                                        <CPUUsage data={props.cpudata}/>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="w-1/2 h-full relative">
                        <div class="w-[80%] h-1/2 flex justify-end items-end pl-[0.5vw]">
                            <Butterfly />
                        </div>
                        <div class="absolute top-0 left-0 w-full h-full flex flex-col space-y-[1.5vh]">
                            <div class="space-x-[1.5vh] items-end flex w-full h-[70%]">
                                <UserCount isMobile={false} count={props.userCount} />
                                <FilesHosted isMobile={false} count={props.filesHosted} />
                                <ContactMe />
                            </div>
                            <div class="justify-center items-center flex flex-col w-full h-[30%] rounded-[1.5vh] bg-[#242424] pt-[2.5vh] p-[2vh]" style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);">
                                <p class="text-white font-semibold text-[2vh]">Space Used (GB)</p>
                                <GraphComponent GraphData={props.spaceUsed}/>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </DesktopTemplate>
    )
}

export default DesktopHome;