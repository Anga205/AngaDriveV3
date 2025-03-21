import type { Component } from 'solid-js';
import { onMount } from 'solid-js';
import { Chart, Title, Tooltip, Colors, ArcElement } from 'chart.js';
import { Doughnut } from 'solid-chartjs';


const CircularProgress: Component = () => {

    onMount(() => {
        Chart.register(Title, Tooltip, Colors, ArcElement);
    });

    const chartData = {
        labels: ['Completed', 'Remaining'],
        datasets: [
            {
                data: [70, 30], // Example: 70% completed, 30% remaining
                backgroundColor: ['#0000ff', '#e0e0e0'], // Colors for the segments
                borderWidth: 0, // Remove border
            },
        ],
    };

    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        cutout: '80%', // Makes it look like a circular progress bar
        plugins: {
            tooltip: {
                enabled: false, // Disable tooltip for simplicity
            },
        },
    };

    return (
        <div class="w-11/12 h-9/12 overflow-hidden">
            <Doughnut data={chartData} options={chartOptions} />
        </div>
    );
};

export default CircularProgress;