import { Component } from "solid-js";
import X from "lucide-solid/icons/x"
import FileText from "lucide-solid/icons/file-text"
import LayoutGrid from "lucide-solid/icons/layout-grid"
import DatabaseZap from "lucide-solid/icons/database-zap"
import File from "lucide-solid/icons/file"
import Github from "lucide-solid/icons/github"
import AlignJustify from "lucide-solid/icons/align-justify"
import ArrowDownToLine from "lucide-solid/icons/arrow-down-to-line"
import RotateCcw from "lucide-solid/icons/rotate-ccw"
import House from "lucide-solid/icons/house"    
import Butterfly from "./butterfly.svg"
import Anga from "./anga.svg"
import OctagonX from "lucide-solid/icons/octagon-x"

const FileTextSVG: Component<{ class?: string }> = (props) => {
    return (
        <FileText color={"white"} class={`w-full h-auto ${props.class || ''}`} stroke-width={0.7} />
    )
}

const CollectionSVG: Component = () => {
    return (
        <LayoutGrid color={"white"} class="w-full h-auto"/>
    )
}

const DatabaseZapSVG: Component = () => {
    return (
        <DatabaseZap color="#2ec9ff" style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);" class="rounded-[1.3vh] bg-[#404040] w-full h-full p-[1.4vh]" stroke-width={1.3}/>
    )
}

const FileSVG: Component<{ class?: string }> = (props) => {
    return (
        <File class={`lucide lucide-file w-full h-auto ${props.class || ''}`} color="white"/>
    )
}

const GitHubSVG: Component = () => {
    return (
        <Github class="w-full h-auto" color="white"/>
    )
}

const HamburgerSVG: Component = () => {
    return (
        <AlignJustify class="w-full h-auto" color="white"/>
    )
}

const HomeSVG: Component = () => {
    return (
        <House class="w-full h-auto" color="white"/>
    )
}

const LockSVG: Component = () => {
    return (
        <svg style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#00a63e" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" class="rounded-[1.3vh] bg-[#404040] w-full h-full p-[1.5vh] lucide lucide-lock"><rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
    )
}

const ScanEyeSVG: Component = () => {
    return (
        <svg style="box-shadow: inset -4px 4px 6px rgba(0, 0, 0, 0.3);" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#8200db" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-scan-eye rounded-[1.3vh] bg-[#404040] w-full h-full p-[1.4vh]"><path d="M3 7V5a2 2 0 0 1 2-2h2"/><path d="M17 3h2a2 2 0 0 1 2 2v2"/><path d="M21 17v2a2 2 0 0 1-2 2h-2"/><path d="M7 21H5a2 2 0 0 1-2-2v-2"/><circle cx="12" cy="12" r="1"/><path d="M18.944 12.33a1 1 0 0 0 0-.66 7.5 7.5 0 0 0-13.888 0 1 1 0 0 0 0 .66 7.5 7.5 0 0 0 13.888 0"/></svg>
    )
}

const UnlockSVG: Component = () => {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-lock-open"><rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 9.9-1"/></svg>
    )
}

const UserSVG: Component = () => {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-user w-full h-auto"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
    )
}

const UploadSVG: Component = () => {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-arrow-up-to-line-icon lucide-arrow-up-to-line"><path d="M5 3h14"/><path d="m18 13-6-6-6 6"/><path d="M12 7v14"/></svg>
    )
}

const InfoSVG: Component = () => {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-info-icon lucide-info w-full h-auto"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/></svg>
    )
}

const ErrorSVG: Component = () => {
    return (
        <OctagonX />
    )
}

const EyeSVG: Component = () => {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-eye-icon lucide-eye"><path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0"/><circle cx="12" cy="12" r="3"/></svg>
    )
}

const CopySVG: Component = () => {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-copy-icon lucide-copy"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>
    )
}

const BinSVG: Component = () => {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-trash2-icon lucide-trash-2"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/><line x1="10" x2="10" y1="11" y2="17"/><line x1="14" x2="14" y1="11" y2="17"/></svg>
    )
}

const DownloadSVG: Component = () => {
    return (
        <ArrowDownToLine class="w-full h-auto"/>
    )
}

const RefreshSVG: Component = () => {
    return (
        <RotateCcw class="w-full h-auto"/>
    )
}

const CrossSVG = () => (
    <X />
)

export {Butterfly, CollectionSVG, DatabaseZapSVG, FileSVG, GitHubSVG, HamburgerSVG, HomeSVG, LockSVG, ScanEyeSVG, UnlockSVG, UserSVG, Anga, UploadSVG, InfoSVG, ErrorSVG, EyeSVG, CopySVG, BinSVG, DownloadSVG, FileTextSVG, RefreshSVG, CrossSVG}