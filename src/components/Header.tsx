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
        <div class="w-full flex items-center h-[9.5vh] justify-between">
            <div>
                <p class="text-white font-extrabold font-sans text-[4vh]">
                    Drive V3
                </p>
                <p class="text-white font-normal text-[1.8vh]">
                    &OpenCurlyDoubleQuote;<em>{randomQuote.text}</em>&CloseCurlyDoubleQuote; — {randomQuote.author}
                </p>
            </div>
            <a class="text-blue-500 font-semibold hover:underline cursor-pointer text-[1.65vh]" href="https://github.com/shakirth-anisha">
                Designed by Anisha
            </a>
        </div>
    )
}

const MobileHeader: Component = () => {
    const generateDistinctColor = (existingColors: string[]): string => {
        let color: string;
        do {
            color = `#${Math.floor(Math.random() * 16777215).toString(16).padStart(6, '0')}`;
        } while (
            existingColors.some(existingColor => {
                const colorDiff = (c1: string, c2: string) => {
                    const r1 = parseInt(c1.slice(1, 3), 16);
                    const g1 = parseInt(c1.slice(3, 5), 16);
                    const b1 = parseInt(c1.slice(5, 7), 16);
                    const r2 = parseInt(c2.slice(1, 3), 16);
                    const g2 = parseInt(c2.slice(3, 5), 16);
                    const b2 = parseInt(c2.slice(5, 7), 16);
                    return Math.sqrt((r1 - r2) ** 2 + (g1 - g2) ** 2 + (b1 - b2) ** 2);
                };
                return colorDiff(color, existingColor) < 100; // Ensure sufficient color difference
            }) || 
            parseInt(color.slice(1, 3), 16) < 50 || // Avoid very dark red
            parseInt(color.slice(3, 5), 16) < 50 || // Avoid very dark green
            parseInt(color.slice(5, 7), 16) < 50    // Avoid very dark blue
        );
        return color;
    };

    const colors: string[] = [];
    for (let i = 0; i < 3; i++) {
        colors.push(generateDistinctColor(colors));
    }

    interface Quote {
        text: string;
        author: string;
    }

    const typedQuotes: Quote[] = quotes as Quote[];
    const randomQuote = typedQuotes[Math.floor(Math.random() * typedQuotes.length)];

    return (
        <div>
            <p class="text-white w-full text-center text-[10vw] font-semibold">
                <span style={{ color: colors[0] }}>Anga</span>
                <span style={{ color: colors[1] }}>Drive</span>
                <span style={{ color: colors[2] }}>V3</span>
            </p>
            <p class="text-white text-center text-[3vw] mt-2 w-full px-2">
                &OpenCurlyDoubleQuote;<em>{randomQuote.text}</em>&CloseCurlyDoubleQuote;<br/> — {randomQuote.author}
            </p>
        </div>
    );
};

export {Header, MobileHeader}