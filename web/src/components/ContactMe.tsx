import { Component } from "solid-js";
import { createSignal } from "solid-js";
import emailjs, { EmailJSResponseStatus } from "@emailjs/browser";

const ContactMe: Component = () => {
    const [email, setEmail] = createSignal("");
    const [message, setMessage] = createSignal("");

    const isEmailValid = (email: string) => {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return emailRegex.test(email);
    };

    const handleSendEmail = () => {
        const formDataWithTimeStamp = {
            contact: email(),
            message: message(),
            timeStamp: new Date().toLocaleString('en-IN', {
                day: 'numeric',
                month: 'short',
                year: 'numeric',
                hour: 'numeric',
                minute: 'numeric',
                hour12: true,
                timeZone: 'Asia/Kolkata'
            })
        };

        emailjs.send(
            "service_3ys6akq",
            "template_gervg6v",
            formDataWithTimeStamp,
            "1JMl4awfqw7WioBaX"
        ).then(
            (response: EmailJSResponseStatus) => {
                console.log("SUCCESS!", response);
                alert("Email sent successfully!");
            },
            (error: Error) => {
                console.log("FAILED...", error);
                alert("Failed to send email.");
            }
        );
    }

    return (
        <div class="bg-[#242424] rounded-xl flex flex-col justify-center items-center p-4 space-y-4 shadow-inner shadow-black/30 
            w-2/3 aspect-[2/1] sm:w-[44%] sm:h-[38vh] sm:rounded-[1.5vh] sm:p-[2vh] sm:space-y-[2vh]">
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
                class="bg-blue-600 text-white hover:bg-blue-800 disabled:bg-blue-950 font-semibold text-[3vw] p-[1vh] rounded-[0.5vh] w-full 
                sm:text-[1.65vh] sm:p-[0.8vh] sm:rounded-[0.3vh]"
                disabled={!email().trim() || !message().trim() || !isEmailValid(email())}
                onClick={handleSendEmail}>
                Send
            </button>
        </div>
    );
};

export default ContactMe;
