import { Accessor, Component, createEffect, createSignal } from "solid-js";
import Dialog from '@corvu/dialog';
import { toast } from "solid-toast";
import { useWebSocket } from "@/Websockets";
import bcrypt from "bcryptjs";
import { isEmailValid, isPasswordValid } from "../validators";

const AccountDetails: Component<{email: Accessor<string>; setEmail: (email: string) => void; displayName: Accessor<string>; setDisplayName: (name: string) => void; class?: string}> = (props) => {
    const [open, setOpen] = createSignal(false);
    const [tempDisplayName, setTempDisplayName] = createSignal(props.displayName());
    const [tempEmail, setTempEmail] = createSignal(props.email());
    const [tempNewPassword, setTempNewPassword] = createSignal("");
    const [currentAuthPassword, setCurrentAuthPassword] = createSignal("");
    const [hasPendingChanges, setHasPendingChanges] = createSignal(false);

    const { socket: getSocket, status: socketStatus } = useWebSocket();

    createEffect(() => {
        const dnChanged = tempDisplayName() !== props.displayName();
        const emailChanged = tempEmail() !== props.email();
        const pwChanged = tempNewPassword() !== "";
        setHasPendingChanges(dnChanged || emailChanged || pwChanged);
    });

    const resetDialogFormState = () => {
        setTempDisplayName(props.displayName());
        setTempEmail(props.email());
        setTempNewPassword("");
        setCurrentAuthPassword("");
        // hasPendingChanges will be updated by the createEffect
    };

    const handleSaveChanges = async () => {
        const currentSocket = getSocket();

        if (socketStatus() === "connecting" || socketStatus() === "reconnecting") {
            toast.error("WebSocket is connecting. Please wait and try again.");
            return;
        }

        if (!currentSocket || currentSocket.readyState !== WebSocket.OPEN) {
            toast.error("WebSocket is not connected. Please try again later.");
            return;
        }

        if (!hasPendingChanges()) {
            toast.error("No changes to save.");
            setOpen(false);
            return;
        }

        if (!currentAuthPassword()) {
            toast.error("Please enter your current password to authenticate changes.");
            return;
        }

        const initialEmailForAuth = props.email(); // Email at the time of opening dialog
        const authPasswordForRequests = currentAuthPassword();

        let anyChangeSucceeded = false;
        let allOperationsAttemptedAndSuccessful = true; // Assume success until a failure or skipped operation

        const sendUpdateRequest = (
            requestType: string,
            payload: any,
            successMessage: string,
            successCallback?: (data: any) => void
        ): Promise<boolean> => {
            return new Promise<boolean>((resolve) => {
                if (!currentSocket || currentSocket.readyState !== WebSocket.OPEN) {
                    toast.error(`WebSocket connection lost before sending ${requestType}.`);
                    resolve(false);
                    return;
                }

                let messageHandler: ((event: MessageEvent) => void) | null = null;
                let errorHandler: ((event: Event) => void) | null = null;
                let closeHandler: (() => void) | null = null;

                const cleanupListeners = () => {
                    if (messageHandler && currentSocket) currentSocket.removeEventListener('message', messageHandler);
                    if (errorHandler && currentSocket) currentSocket.removeEventListener('error', errorHandler);
                    if (closeHandler && currentSocket) currentSocket.removeEventListener('close', closeHandler);
                    messageHandler = null;
                    errorHandler = null;
                    closeHandler = null;
                };

                messageHandler = (event: MessageEvent) => {
                    try {
                        const response = JSON.parse(event.data);
                        if (response.type === `${requestType}_response`) {
                            cleanupListeners();
                            if (response.data.error) {
                                toast.error(
                                    response.data.error === "record not found" || response.data.error === "invalid credentials"
                                        ? "Authentication failed. Check current password."
                                        : response.data.error.charAt(0).toUpperCase() + response.data.error.slice(1)
                                );
                                resolve(false);
                            } else {
                                toast.success(successMessage);
                                if (successCallback) successCallback(response.data);
                                anyChangeSucceeded = true;
                                resolve(true);
                            }
                        }
                    } catch (err) {
                        cleanupListeners();
                        console.error(`Failed to parse ${requestType} response:`, err);
                        toast.error("Received an invalid response from server.");
                        resolve(false);
                    }
                };

                errorHandler = (error: Event) => {
                    cleanupListeners();
                    console.error(`WebSocket error during ${requestType}:`, error);
                    toast.error(`An error occurred while updating ${requestType.replace("_", " ")}.`);
                    resolve(false);
                };

                closeHandler = () => {
                    if (messageHandler) { // Only if not already cleaned up
                        cleanupListeners();
                        console.warn(`WebSocket connection closed during ${requestType} attempt.`);
                        toast.error(`Update for ${requestType.replace("_", " ")} failed: Connection lost.`);
                        resolve(false);
                    }
                };

                currentSocket.addEventListener('message', messageHandler);
                currentSocket.addEventListener('error', errorHandler);
                currentSocket.addEventListener('close', closeHandler);

                currentSocket.send(JSON.stringify({ type: requestType, data: payload }));
            });
        };

        // --- Change Display Name ---
        if (tempDisplayName() !== props.displayName()) {
            if (tempDisplayName().length < 3 || tempDisplayName().length > 64) {
                toast.error("Display name must be between 3 and 64 characters.");
                allOperationsAttemptedAndSuccessful = false;
            } else {
                const success = await sendUpdateRequest(
                    "change_display_name",
                    {
                        email: initialEmailForAuth,
                        password: authPasswordForRequests,
                        display_name: tempDisplayName(),
                    },
                    "Display name updated!",
                    () => {
                        localStorage.setItem("display_name", tempDisplayName());
                        props.setDisplayName(tempDisplayName());
                    }
                );
                if (!success) allOperationsAttemptedAndSuccessful = false;
            }
        }

        // --- Change Email ---
        if (tempEmail() !== props.email() && allOperationsAttemptedAndSuccessful) { // Proceed if previous steps were fine
            if (!isEmailValid(tempEmail())) {
                toast.error("Please enter a valid new email address.");
                allOperationsAttemptedAndSuccessful = false;
            } else {
                const success = await sendUpdateRequest(
                    "change_email",
                    {
                        old_email: initialEmailForAuth,
                        password: authPasswordForRequests,
                        new_email: tempEmail(),
                    },
                    "Email updated!",
                    () => {
                        localStorage.setItem("email", tempEmail());
                        props.setEmail(tempEmail());
                    }
                );
                if (!success) allOperationsAttemptedAndSuccessful = false;
            }
        }

        // --- Change Password ---
        if (tempNewPassword() && allOperationsAttemptedAndSuccessful) { // Proceed if previous steps were fine
            if (!isPasswordValid(tempNewPassword())) {
                toast.error("New password must be between 3 and 64 characters.");
                allOperationsAttemptedAndSuccessful = false;
            } else {
                const salt = bcrypt.genSaltSync(10);
                const hashedPassword = bcrypt.hashSync(tempNewPassword(), salt);
                const success = await sendUpdateRequest(
                    "change_password",
                    {
                        email: localStorage.getItem("email") || initialEmailForAuth, // Use potentially updated email
                        old_password: authPasswordForRequests,
                        new_password_hashed: hashedPassword,
                    },
                    "Password updated!",
                    () => localStorage.setItem("password", tempNewPassword())
                );
                if (!success) allOperationsAttemptedAndSuccessful = false;
            }
        }

        if (anyChangeSucceeded) {
            if (allOperationsAttemptedAndSuccessful) {
                setOpen(false); // Close dialog only if all attempted operations were successful
                // resetDialogFormState() is called by onOpenChange when dialog closes
            } else {
                toast.error("Some changes may not have been saved. Please review.");
                // Refresh temp states from localStorage to reflect partial successes
                setTempDisplayName(localStorage.getItem("display_name") || props.displayName());
                setTempEmail(localStorage.getItem("email") || props.email());
                // Don't clear tempNewPassword or currentAuthPassword if a password change was attempted and failed
            }
        } else if (hasPendingChanges()) { // No change succeeded but changes were attempted
            toast.error("Failed to save changes. Please check your current password and details.");
        }
    };

    return (
        <Dialog open={open()} onOpenChange={(newOpen) => {
            setOpen(newOpen);
            if (!newOpen) {
                resetDialogFormState();
            } else {
                // Ensure form is reset to current prop values when opening
                setTempDisplayName(props.displayName());
                setTempEmail(props.email());
                setTempNewPassword("");
                setCurrentAuthPassword("");
            }
        }}>
            <div class={`bg-gray-950 border border-gray-700 rounded-md w-full p-[2vh] text-white flex flex-col shadow-md ${props.class ?? ''}`}>
                <p class="font-semibold text-[2.5vh] mb-[2vh]">Account Details</p>
                <div class="mb-[1vh]">
                    <p class="text-gray-500 text-[1.5vh] uppercase tracking-wider">Display Name:</p>
                    <p class="text-[2vh]">{props.displayName()}</p>
                </div>
                <div class="mb-[1vh]">
                    <p class="text-gray-500 text-[1.5vh] uppercase tracking-wider">Email:</p>
                    <p class="text-[2vh] truncate">{props.email()}</p>
                </div>
                <div class="mb-[2vh]">
                    <p class="text-gray-500 text-[1.5vh] uppercase tracking-wider">Password:</p>
                    <p class="text-[2vh]">************</p>
                </div>
                <Dialog.Trigger class="bg-blue-600 hover:bg-blue-700 text-white font-semibold py-[1vh] rounded mt-auto transition-colors duration-200 text-[1.5vh]">
                    Edit Account
                </Dialog.Trigger>
            </div>
            <Dialog.Portal>
                <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
                <Dialog.Content class="flex z-50 justify-center flex-col fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-neutral-800 p-6 rounded-md shadow-lg text-white w-[clamp(300px,50vw,500px)]">
                    <Dialog.Label class="text-xl font-semibold mb-2 text-center">Edit Account Details</Dialog.Label>
                    <p class="mb-1 text-sm text-neutral-400">Change your details below.</p>
                    <p class="mb-4 text-sm text-neutral-400">Enter your <strong class="text-neutral-300">current password</strong> to save any changes.</p>

                    <label for="displayNameInput" class="text-sm text-neutral-300 mb-1 mt-2">Display Name:</label>
                    <input
                        id="displayNameInput"
                        type="text"
                        class="w-full p-3 rounded bg-neutral-700 text-[1.5vh] placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
                        placeholder="New Display Name"
                        value={tempDisplayName()}
                        onInput={e => setTempDisplayName(e.currentTarget.value)}
                        autocomplete="off"
                    />
                    <label for="emailInput" class="text-sm text-neutral-300 mb-1 mt-3">Email ID:</label>
                    <input
                        id="emailInput"
                        type="email"
                        class="w-full p-3 rounded bg-neutral-700 text-[1.5vh] placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
                        placeholder="New Email ID"
                        value={tempEmail()}
                        onInput={e => setTempEmail(e.currentTarget.value)}
                        autocomplete="off"
                    />
                    <label for="newPasswordInput" class="text-sm text-neutral-300 mb-1 mt-3">New Password (optional):</label>
                    <input
                        id="newPasswordInput"
                        type="password"
                        class="w-full p-3 rounded bg-neutral-700 text-[1.5vh] placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-blue-400"
                        placeholder="Leave blank to keep current password"
                        value={tempNewPassword()}
                        onInput={e => setTempNewPassword(e.currentTarget.value)}
                        autocomplete="new-password"
                    />

                    <div class={`${hasPendingChanges() ? 'mt-3' : 'hidden'}`}>
                        <label for="currentPasswordInput" class="text-sm text-red-400 mb-1 font-semibold">Current Password (required to save):</label>
                        <input
                            id="currentPasswordInput"
                            type="password"
                            class="w-full p-3 rounded bg-neutral-700 text-[1.5vh] placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-red-400"
                            placeholder="Enter Current Password"
                            value={currentAuthPassword()}
                            onInput={e => setCurrentAuthPassword(e.currentTarget.value)}
                            autocomplete="current-password"
                        />
                    </div>

                    <div class="flex justify-end space-x-3 mt-6">
                        <Dialog.Close class="bg-neutral-600 hover:bg-neutral-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200">
                            Cancel
                        </Dialog.Close>
                        <button
                            class={`bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2 px-4 rounded transition-colors duration-200 ${(!hasPendingChanges() || !currentAuthPassword() || socketStatus() !== 'connected') ? 'opacity-50 cursor-not-allowed' : ''}`}
                            onClick={handleSaveChanges}
                            disabled={!hasPendingChanges() || !currentAuthPassword() || socketStatus() !== 'connected'}
                        >
                            Save Changes
                        </button>
                    </div>
                </Dialog.Content>
            </Dialog.Portal>
        </Dialog>
    )
}

export default AccountDetails;