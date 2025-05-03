import type { Component } from 'solid-js';
import { onMount, onCleanup, createSignal } from 'solid-js';
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
                data: [props.data.used_ram, props.data.available_ram], 
                backgroundColor: ['#04a9e7', '#404040'], 
                borderWidth: 0, 
            },
        ],
    });

    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        cutout: '85%',
        plugins: {
            tooltip: {
                enabled: false,
            },
        },
        animation: {
            duration: 0,
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

const CPUUsageRough: Component<{data: CPUData, smoothPercentage: number}> = (props) => {

    onMount(() => {
        Chart.register(Title, Tooltip, Colors, ArcElement);
    });

    const chartData = () => ({
        labels: ['Completed', 'Remaining'],
        datasets: [
            {
                data: [props.smoothPercentage, 100 - props.smoothPercentage], 
                backgroundColor: ['#04a9e7', '#404040'],
                borderWidth: 0,
            },
        ],
    });

    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        cutout: '85%',
        plugins: {
            tooltip: {
                enabled: false,
            },
        },
        animation: {
            duration: 0,
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

const CPUUsage: Component<{data: CPUData}> = (props) => {

    const [smoothPercentage, setSmoothPercentage] = createSignal(0);
    const smoothFactor = 0.01;
    const smooth = (newValue: number) => {
        setSmoothPercentage((prev: number) => {
            const diff = newValue - prev;
            if (Math.abs(diff) < 0.01) {
                return newValue;
            }
            return prev + diff * smoothFactor;
        });
    };
    const interval = setInterval(() => {
        smooth(props.data.cpu_usage);
    }, 1);
    onCleanup(() => clearInterval(interval));

    return (
        <CPUUsageRough 
            data = {props.data}
            smoothPercentage = {smoothPercentage()}
        />
    )
}

export {RAMUsage, CPUUsage};