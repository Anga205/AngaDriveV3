import { Component } from "solid-js";
import { createSignal } from "solid-js";
import emailjs from "@emailjs/browser";
import toast from "solid-toast";

const ContactMe: Component = () => {
    const [email, setEmail] = createSignal(localStorage.getItem('email') || "");
    const [message, setMessage] = createSignal("");

    const isEmailValid = (email: string) => {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return emailRegex.test(email);
    };

    const handleSendEmail = () => {
        if (!email().trim() || !message().trim() || !isEmailValid(email())) {
            if (!email().trim()) {
                toast.error("Email is required.");
                return
            }
            if (!message().trim()) {
                toast.error("Message is required.");
                return
            }
            if (!isEmailValid(email())) {
                toast.error("Invalid email format.");
                return
            }
        } else {
            const formDataWithTimeStamp = {
                contact: email(),
                message: message(),
                time: new Date().toLocaleString('en-IN', {
                    day: 'numeric',
                    month: 'short',
                    year: 'numeric',
                    hour: 'numeric',
                    minute: 'numeric',
                    hour12: true,
                    timeZone: 'Asia/Kolkata'
                })
            };
    
            const emailPromise = emailjs.send(
                "service_3ys6akq",
                "template_gervg6v",
                formDataWithTimeStamp,
                "1JMl4awfqw7WioBaX"
            );
    
            toast.promise(emailPromise, {
                loading: "Sending email...",
                success: "Email sent successfully!",
                error: "Failed to send email.",
            });
    
            emailPromise.then(() => {
                setEmail(""); // Clear the email input
                setMessage(""); // Clear the message input
                }
            );
        }
    }

    const enableButton = () => {
        return (!email().trim() || !message().trim() || !isEmailValid(email()))
    }

    return (
        <div class="bg-[#242424] rounded-xl flex flex-col justify-center items-center p-4 space-y-4 shadow-inner shadow-black/30 
            w-2/3 aspect-2/1 sm:w-[44%] sm:h-[38vh] sm:rounded-[1.5vh] sm:p-[2vh] sm:space-y-[2vh]">
            <p class="text-white font-semibold text-[4vw] sm:text-[2vh]">Send me an Email</p>
            <input 
                placeholder="Your Email" 
                class="text-[3vw] w-full bg-[#323232] placeholder-[#959595] p-[1vh] rounded-[0.5vh] 
                sm:text-[1.65vh] sm:p-[0.8vh] sm:rounded-[0.3vh] text-white" 
                value={email()} 
                onInput={(e) => setEmail(e.currentTarget.value)} 
            />
            <textarea 
                placeholder="Your Message" 
                class="text-[3vw] w-full bg-[#323232] placeholder-[#959595] p-[1vh] rounded-[0.5vh] h-full 
                sm:text-[1.65vh] sm:p-[0.8vh] sm:rounded-[0.3vh] text-white" 
                value={message()} 
                onInput={(e) => setMessage(e.currentTarget.value)} 
            />
            <button 
                class={`font-semibold text-[3vw] p-[1vh] rounded-[0.5vh] w-full sm:text-[1.65vh] sm:p-[0.8vh] sm:rounded-[0.3vh] text-white ${
                    enableButton()
                        ? "cursor-not-allowed bg-blue-800"
                        : "hover:bg-blue-800 bg-blue-600"
                }`}
                onClick={handleSendEmail}>
                Send
            </button>
        </div>
    );
};

export default ContactMe;
