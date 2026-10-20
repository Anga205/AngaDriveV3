import { Component, createSignal, useContext } from "solid-js";
import Navbar from "@/components/Navbar";
import { AppContext } from "@/Context";
import { formatFileSize } from "@/library/functions";
import { GitHubSVG } from "@/assets/SvgFiles";
import AccountDetails from "../shared/components/AccountDetails";
import { DangerZone, UserStat } from "../shared/components/DangerZone";

const MobileAccountManager: Component<{logout: () => void}> = (props) => {
    const [email, setEmail] = createSignal(localStorage.getItem("email") || "{email}");
    const [displayName, setDisplayName] = createSignal(localStorage.getItem("display_name") || "{display_name}");
    const ctx = useContext(AppContext)!;

    return (
        <div class="w-full h-dvh bg-black flex flex-col pb-16">
            <Navbar CurrentPage="Account" Type="mobile" />
            <div class="overflow-y-auto p-4 space-y-4 items-center justify-center h-full">
                <AccountDetails email={email} setEmail={setEmail} displayName={displayName} setDisplayName={setDisplayName} />
                <div class="grid grid-cols-2 gap-4">
                    <UserStat title="Space Used" value={formatFileSize(ctx.files().reduce((sum, file) => sum + file.file_size, 0))} class="col-span-2"/>
                    <UserStat title="Files Hosted" value={ctx.files().length.toString()} />
                    <UserStat title="Collections" value={ctx.userCollections().size.toString()}/>
                </div>
                <a
                    href="https://github.com/Anga205/AngaDriveV3"
                    target="_blank"
                    rel="noreferrer"
                    class="w-full flex items-center justify-center gap-2 bg-neutral-900 border border-neutral-700 rounded-md p-[1.5vh] text-white font-semibold hover:bg-neutral-800 transition-colors"
                >
                    <div class="w-5 h-5">
                        <GitHubSVG />
                    </div>
                    <span>View on GitHub</span>
                </a>
                <DangerZone logout={props.logout} />
            </div>
        </div>
    );
}

export default MobileAccountManager;