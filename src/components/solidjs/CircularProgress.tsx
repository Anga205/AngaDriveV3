import type { Component } from 'solid-js';
import { onMount } from 'solid-js';
import { Chart, Title, Tooltip, Colors, ArcElement } from 'chart.js';
import { Doughnut } from 'solid-chartjs';


const RAMUsage: Component = () => {

    onMount(() => {
        Chart.register(Title, Tooltip, Colors, ArcElement);
    });

    const chartData = {
        labels: ['Completed', 'Remaining'],
        datasets: [
            {
                data: [900, 8000], // Example: 70% completed, 30% remaining
                backgroundColor: ['#04a9e7', '#404040'], // Colors for the segments
                borderWidth: 0, // Remove border
            },
        ],
    };

    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        cutout: '85%', // Makes it look like a circular progress bar
        plugins: {
            tooltip: {
                enabled: false, // Disable tooltip for simplicity
            },
        },
    };

    return (
        <div class="w-7/12 h-9/12 overflow-hidden relative flex items-center">
            <Doughnut data={chartData} options={chartOptions} />
            <div class="top-0 left-0 z-10 flex flex-col items-center justify-center w-full h-full absolute">
                <div class="flex flex-col items-center justify-center">
                    <p class="text-white text-[3vh]">901.82 MB</p>
                    <div class="bg-white w-full h-[1px]"/>
                    <p class="text-white text-[3vh]">7.89 GB</p>
                </div>
            </div>
        </div>
    );
};

const CPUUsage: Component = () => {

    onMount(() => {
        Chart.register(Title, Tooltip, Colors, ArcElement);
    });

    const chartData = {
        labels: ['Completed', 'Remaining'],
        datasets: [
            {
                data: [61, 39], // Example: 70% completed, 30% remaining
                backgroundColor: ['#04a9e7', '#404040'], // Colors for the segments
                borderWidth: 0, // Remove border
            },
        ],
    };

    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        cutout: '85%', // Makes it look like a circular progress bar
        plugins: {
            tooltip: {
                enabled: false, // Disable tooltip for simplicity
            },
        },
    };

    return (
        <div class="w-7/12 h-9/12 overflow-hidden relative flex items-center">
            <Doughnut data={chartData} options={chartOptions} />
            <div class="top-0 left-0 z-10 flex flex-col items-center justify-center w-full h-full absolute">
                <div class="flex flex-col items-center justify-center">
                    <p class="text-white text-[5vh]">61%</p>
                </div>
            </div>
        </div>
    );
};

export {RAMUsage, CPUUsage};