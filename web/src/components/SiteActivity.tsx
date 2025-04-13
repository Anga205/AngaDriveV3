import type { Component } from 'solid-js';
import { onMount } from 'solid-js';
import { Chart, Title, Tooltip, Colors } from 'chart.js'
import { Line } from 'solid-chartjs'

const SiteActivity: Component = () => {
    /**
     * You must register optional elements before using the chart,
     * otherwise you will have the most primitive UI
     */
    onMount(() => {
        Chart.register(Title, Tooltip, Colors)
    })

    const chartData = {
        labels: ['Mar 13', 'Mar 14', 'Mar 15', 'Mar 16', 'Mar 17', 'Mar 18', 'Mar 19'],
        datasets: [
            {
                label: 'Database Reads',
                data: [124, 83, 46, 36, 64, 54, 70],
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
                beginAtZero: true
            }
        }
    }

    return (
        <div class="w-full h-full">
            <Line data={chartData} options={chartOptions} />
        </div>
    )
}

export default SiteActivity