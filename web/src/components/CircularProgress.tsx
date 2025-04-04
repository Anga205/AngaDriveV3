import type { Component } from 'solid-js';
import { onMount } from 'solid-js';
import { Chart, Title, Tooltip, Colors, ArcElement } from 'chart.js';
import { Doughnut } from 'solid-chartjs';
import { CPUData, RAMData } from '../types/types';

const RAMUsage: Component<{data: RAMData}> = (props) => {

    onMount(() => {
        Chart.register(Title, Tooltip, Colors, ArcElement);
    });

    const chartData = () => ({
        labels: ['Completed', 'Remaining'],
        datasets: [
            {
                data: [props.data.used_ram, props.data.available_ram], // Example: 70% completed, 30% remaining
                backgroundColor: ['#04a9e7', '#404040'], // Colors for the segments
                borderWidth: 0, // Remove border
            },
        ],
    });

    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        cutout: '85%', // Makes it look like a circular progress bar
        plugins: {
            tooltip: {
                enabled: false, // Disable tooltip for simplicity
            },
        },
        animation: {
            duration: 0, // Set animation duration to zero
        },
    };

    return (
        <div class="w-7/12 h-9/12 overflow-hidden relative flex items-center">
            <Doughnut data={chartData()} options={chartOptions} />
            <div class="top-0 left-0 z-10 flex flex-col items-center justify-center w-full h-full absolute">
                <div class="flex flex-col items-center justify-center">
                    <p class="text-white text-[3.5vw] md:text-[1.5vw]">{(props.data.used_ram / (1024 ** 3)).toFixed(2)} GB</p>
                    <div class="bg-white w-full h-[1px]"/>
                    <p class="text-white text-[3.5vw] md:text-[1.5vw]">{(props.data.total_ram/ (1024**3)).toFixed(2)} GB</p>
                </div>
            </div>
        </div>
    );
};

const CPUUsage: Component<{data: CPUData}> = (props) => {

    onMount(() => {
        Chart.register(Title, Tooltip, Colors, ArcElement);
    });

    const chartData = () => ({
        labels: ['Completed', 'Remaining'],
        datasets: [
            {
                data: [props.data.cpu_usage, 100 - props.data.cpu_usage], // Example: 70% completed, 30% remaining
                backgroundColor: ['#04a9e7', '#404040'], // Colors for the segments
                borderWidth: 0, // Remove border
            },
        ],
    });

    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        cutout: '85%', // Makes it look like a circular progress bar
        plugins: {
            tooltip: {
                enabled: false, // Disable tooltip for simplicity
            },
        },
        animation: {
            duration: 0, // Set animation duration to zero
        },
    };

    return (
        <div class="w-7/12 h-9/12 overflow-hidden relative flex items-center">
            <Doughnut data={chartData()} options={chartOptions} />
            <div class="top-0 left-0 z-10 flex flex-col items-center justify-center w-full h-full absolute">
                <div class="flex flex-col items-center justify-center">
                    <p class="text-white text-[6vw] md:text-[2.5vw]">{props.data.cpu_usage.toFixed(1)}%</p>
                </div>
            </div>
        </div>
    );
};

export {RAMUsage, CPUUsage};