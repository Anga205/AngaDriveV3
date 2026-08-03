import { Component, Accessor } from "solid-js";

const UserCount: Component<{isMobile: boolean; count: Accessor<number>}> = (props) => {
    return (
        <div class={`bg-[#242424] ${props.isMobile?'grow rounded-xl p-[1vw]':'h-[20vh] w-[28%] p-[1vh] rounded-[1.5vh]'} flex flex-col items-center justify-center`} style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);">
            <p class={props.isMobile?"text-center font-black text-white text-[5vw]":"text-white font-semibold text-[0.9vw]"}>Users</p>
            <p class={props.isMobile?"text-center font-black text-white text-[6vw]":"text-white font-semibold text-[5vw]"}>{props.count()}</p>
        </div>
    );
}

const FilesHosted: Component<{ isMobile: boolean; count: Accessor<number> }> = (props) => {
    return (
        <div class={`flex justify-center items-center flex-col bg-[#242424] ${props.isMobile?'grow rounded-xl p-[1vw]':'h-[20vh] w-[28%] p-[1vh] rounded-[1.5vh] overflow-hidden'}`} style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);">
            <p class={`text-white text-center ${props.isMobile?'font-black text-[4vw]':'font-semibold text-[0.9vw]'}`}>Files&nbsp;Hosted</p>
            <p class={`text-white text-center ${props.isMobile?'font-black text-[6vw]':'font-semibold text-[4vw]'}`}>{props.count()}</p>
        </div>
    )
}

export { UserCount, FilesHosted };