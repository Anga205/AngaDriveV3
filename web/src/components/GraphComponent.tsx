import type { Component } from 'solid-js';
import { onMount } from 'solid-js';
import { Chart, Title, Tooltip, Colors } from 'chart.js'
import { Line } from 'solid-chartjs'
import { GraphData } from '../types/types';

const GraphComponent: Component<{ GraphData: GraphData }> = (props) => {
    onMount(() => {
        Chart.register(Title, Tooltip, Colors)
    })

    const chartData = {
        labels: props.GraphData.x_axis,
        datasets: [
            {
                label: props.GraphData.label,
                data: props.GraphData.y_axis,
                borderColor: 'rgba(75, 75, 192, 1)',
                backgroundColor: 'rgba(75, 192, 192, 0.2)',
                borderWidth: 1.5
            },
        ],
    }
    const chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        animation: {
            duration: 0,
        },
        scales: {
            x: {
                grid: {
                    color: 'rgba(255, 255, 255, 0.1)',
                },
                ticks: {
                    color: 'rgba(255, 255, 255, 0.3)',
                },
            },
            y: {
                grid: {
                    color: 'rgba(255, 255, 255, 0.1)',
                },
                ticks: {
                    color: 'rgba(255, 255, 255, 0.3)',
                },
                beginAtZero: props.GraphData.beginAtZero,
            }
        }
    }

    return (
        <div class="w-full h-full">
            <Line data={chartData} options={chartOptions} />
        </div>
    )
}

export default GraphComponent