import { Component } from "solid-js";
import quotes from "../assets/quotes.json";

const Header: Component = () => {
    interface Quote {
      text: string;
      author: string;
    }
    
    const typedQuotes: Quote[] = quotes as Quote[];
    const randomQuote = typedQuotes[Math.floor(Math.random() * typedQuotes.length)];
    return (
        <div class="w-full flex items-center h-[9.5vh]">
            <div>
                <p class="text-white font-extrabold font-sans text-[4vh]">
                    Drive V3
                </p>
                <p class="text-white font-normal text-[1.8vh]">
                    &OpenCurlyDoubleQuote;<em>{randomQuote.text}</em>&CloseCurlyDoubleQuote; — {randomQuote.author}
                </p>
            </div>
            <div class="flex-grow"/>
            <a class="text-blue-500 font-semibold hover:underline cursor-pointer text-[1.65vh]" href="https://github.com/shakirth-anisha">
                Designed by Anisha
            </a>
        </div>
    )
}

export default Header