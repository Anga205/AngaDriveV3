import {DefaultsButtons, MobileButtons} from "../components/DefaultsButtons";
import {Header, MobileHeader} from "../components/Header";
import { RAMUsage, CPUUsage } from "../components/CircularProgress";
import GraphComponent from "../components/GraphComponent";
import { Component, onMount, onCleanup, createSignal, Accessor, createEffect } from "solid-js";
import { Butterfly } from "../assets/SvgFiles";
import type { CPUData, GraphData, IncomingData, RAMData, SysInfo } from "../library/types";
import ContactMe from "../components/ContactMe";
import { DesktopTemplate } from "../components/Template";
import { Toaster } from 'solid-toast';
import Navbar from "../components/Navbar";
import { useWebSocket } from "../Websockets";

const UserCount: Component<{isMobile: boolean; count: Accessor<number>}> = (props) => {
    return (
        <div class={`bg-[#242424] ${props.isMobile?'flex-grow rounded-xl p-[1vw]':'h-[20vh] w-[28%] p-[1vh] rounded-[1.5vh]'} flex flex-col items-center justify-center`} style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);">
            <p class={props.isMobile?"text-center font-black text-white text-[5vw]":"text-white font-semibold text-[0.9vw]"}>Users</p>
            <p class={props.isMobile?"text-center font-black text-white text-[6vw]":"text-white font-semibold text-[5vw]"}>{props.count()}</p>
        </div>
    );
}

const FilesHosted: Component<{ isMobile: boolean; count: Accessor<number> }> = (props) => {
    return (
        <div class={`flex justify-center items-center flex-col bg-[#242424] ${props.isMobile?'flex-grow rounded-xl p-[1vw]':'h-[20vh] w-[28%] p-[1vh] rounded-[1.5vh] overflow-hidden'}`} style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);">
            <p class={`text-white text-center ${props.isMobile?'font-black text-[4vw]':'font-semibold text-[0.9vw]'}`}>Files&nbsp;Hosted</p>
            <p class={`text-white text-center ${props.isMobile?'font-black text-[6vw]':'font-semibold text-[4vw]'}`}>{props.count()}</p>
        </div>
    )
}

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
                                <p class="text-white font-semibold text-[2vh]">Site Activity Over Past Week</p>
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

const MobileHome: Component<{ ramdata: RAMData; cpudata: CPUData; siteActivity: Accessor<GraphData>; spaceUsed: Accessor<GraphData>; userCount: Accessor<number>; filesHosted: Accessor<number> }> = (props) => {
    return (
        <div class="relative w-full min-h-screen">
            <div class="h-screen w-full bg-black flex items-center justify-center p-[5%] fixed">
                <div class="w-full opacity-30">
                    <Butterfly/>
                </div>
            </div>
            <div class="absolute top-0 left-0 w-full space-y-[1vh]">
                <Navbar CurrentPage="Home" Type="mobile"/>
                <div class="h-[5vh]"/>
                <MobileHeader />
                <MobileButtons />
                <div class="w-full px-[1.5vh]">
                    <div class="w-full bg-[#242424] aspect-video rounded-xl p-[1vw] opacity-95">
                        <p class="text-center font-black text-white text-[4vw]">Site Activity Over Past Week</p>
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
                    <div class="w-full aspect-[64/27] p-[1vw] bg-[#242424] rounded-xl">
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
    )
}

const HomePage: Component = () => {
    const [isMobile, setIsMobile] = createSignal(window.innerWidth <= 640);
    const [userCount, setUserCount] = createSignal(0);
    const [filesHosted, setFilesHosted] = createSignal(0);

    const handleResize = () => {
        setIsMobile(window.innerWidth <= 640);
    };
    const [systemInformation, setSystemInformation] = createSignal<SysInfo>({
        ram: {
            total_ram: 0,
            used_ram: 0,
            available_ram: 1,
            ram_percent_used: 0,
        },
        cpu: {
            cpu_model_name: '',
            cpu_usage: 0,
        },
    });

    const [spaceUsed, setSpaceUsed] = createSignal<GraphData>({
        x_axis: Array.from({ length: 7 }, (_, i) => {
            const date = new Date();
            date.setDate(date.getDate() - (6 - i));
            return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
        }),
        y_axis: [0, 0, 0, 0, 0, 0, 0],
        label: 'Space Used',
        begin_at_zero: false,
    });

    const [siteActivity, setSiteActivity] = createSignal<GraphData>({
        x_axis: Array.from({ length: 7 }, (_, i) => {
            const date = new Date();
            date.setDate(date.getDate() - (6 - i));
            return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
        }),
        y_axis: [0, 0, 0, 0, 0, 0, 0],
        label: 'Database Reads',
        begin_at_zero: true,
    })

    const { socket: getSocket } = useWebSocket();

    onMount(() => {
        window.addEventListener('resize', handleResize);
    });

    let currentSocket: WebSocket | null = null; // Keep track of the current socket

    const messageHandler = (event: MessageEvent) => {
        try {
            const data = JSON.parse(event.data) as IncomingData;
            if (data.type === 'system_information') {
                setSystemInformation(data.data as SysInfo);
            } else if (data.type === 'graph_data') {
                const graphData = data.data as GraphData;
                if (graphData.label === 'Space Used') {
                    graphData.y_axis = graphData.y_axis.map(bytes => (bytes / (1024 ** 3)));
                                        graphData.label = 'Space Used (GB)';
                    setSpaceUsed(graphData);
                } else if (graphData.label === 'Site Activity' || graphData.label === 'Database Reads') {
                    setSiteActivity(graphData);
                }
            } else if (data.type === 'user_count') {
                setUserCount(data.data as number);
            }
            else if (data.type === 'files_hosted_count') {
                setFilesHosted(data.data as number);
            }
        } catch (error) {
            if (import.meta.env.DEV) {
                console.error('Failed to parse WebSocket message:', error);
            }
        }
    };

    const openHandler = (updatedSocket: WebSocket) => {
        if (updatedSocket.readyState === WebSocket.OPEN) {
            console.log("HomePage.tsx: Socket open, enabling updates.");
            updatedSocket.send(JSON.stringify({ type: 'enable_homepage_updates', data: true }));
        }
    };

    createEffect(() => {
        const newSocket = getSocket(); // Get the latest socket from the signal

        if (newSocket) {



            // Add listeners to the new socket
            newSocket.addEventListener('message', messageHandler);
            const socketOpenHandler = () => openHandler(newSocket);
            newSocket.addEventListener('open', socketOpenHandler);

            if (newSocket.readyState === WebSocket.OPEN) {
                openHandler(newSocket);
            }

            // Cleanup function (called when the effect re-runs or the component unmounts)

            currentSocket = newSocket; // Update the current socket
            onCleanup(() => {
                console.log("HomePage.tsx: Cleaning up socket listeners.");

                // Remove listeners from the *old* socket (if there was one)
                if (currentSocket) {
                    currentSocket.removeEventListener('message', messageHandler);
                    currentSocket.removeEventListener('open', socketOpenHandler);

                    if (currentSocket.readyState === WebSocket.OPEN) {
                        console.log("HomePage.tsx: Disabling updates for old/unmounting socket.");
                        currentSocket.send(JSON.stringify({ type: 'enable_homepage_updates', data: false }));
                    }
                }
            });
        }
    });
      
    onCleanup(() => {
        window.removeEventListener('resize', handleResize);
    });

    return (
        <>
            <title>HomePage | DriveV3</title>
            {
            isMobile() ? (
                <MobileHome cpudata={systemInformation()!.cpu} ramdata={systemInformation()!.ram} siteActivity={siteActivity} spaceUsed={spaceUsed} userCount={userCount} filesHosted={filesHosted} />
            ) : (
                <DesktopHome cpudata={systemInformation()!.cpu} ramdata={systemInformation()!.ram} siteActivity={siteActivity} spaceUsed={spaceUsed} userCount={userCount} filesHosted={filesHosted} />
            )
            }
            <Toaster
            position="top-center"
            gutter={8}
            containerClassName=""
            containerStyle={{}}
            toastOptions={{
                className: '',
                duration: 2000,
                style: {
                background: '#363636',
                color: '#fff',
                },
            }}
            />
        </>
    );
}

export default HomePage;