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
              data: [124, 83, 46, 36, 64, 54, 70],
          },
      ],
  }
  const chartOptions = {
      responsive: true,
      maintainAspectRatio: false,
  }

  return (
      <div class="w-full h-full pb-[2vh]">
          <Line data={chartData} options={chartOptions} />
      </div>
  )
}

export default SiteActivity