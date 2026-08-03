import { Component, createSignal, useContext } from "solid-js";
import { DesktopTemplate } from "@/components/Template";
import { AppContext } from "@/Context";
import { formatFileSize } from "@/library/functions";
import AccountDetails from "../shared/components/AccountDetails";
import { DangerZone, UserStat } from "../shared/components/DangerZone";

const AccountManager: Component<{logout: () => void}> = (props) => {

    const [email, setEmail] = createSignal(localStorage.getItem("email") || "{email}");
    const [displayName, setDisplayName] = createSignal(localStorage.getItem("display_name") || "{display_name}");
    const ctx = useContext(AppContext)!;

    return (
        <DesktopTemplate CurrentPage="Account">
            <div class="w-full h-full flex justify-center items-center space-x-[1vh]">
                <div class="flex flex-col min-w-[20%] space-y-[2vh]">
                    <AccountDetails email={email} setEmail={setEmail} displayName={displayName} setDisplayName={setDisplayName}/>
                </div>
                <div class="min-w-[20%] grid grid-cols-2 grid-rows-2 gap-[1vh]">
                    <UserStat title="Space&nbsp;Used" value={formatFileSize(ctx.files().reduce((sum, file) => sum + file.file_size, 0))} class="col-span-2"/>
                    <UserStat title="Files&nbsp;Hosted" value={ctx.files().length.toString()} />
                    <UserStat title="Collections" value={ctx.userCollections().size.toString()}/>
                    <DangerZone logout={props.logout} class="col-span-2"/>
                </div>
            </div>
        </DesktopTemplate>
    )
}

export default AccountManager;