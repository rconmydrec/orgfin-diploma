<template>
  <div class="balance-chart-container">
    <canvas ref="chartCanvas"></canvas>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, onBeforeUnmount } from 'vue';
import { Chart, registerables } from 'chart.js';

Chart.register(...registerables);

const props = defineProps({
  projectionData: {
    type: Object,
    required: true,
  },
  currentBalance: {
    type: Number,
    required: true,
  },
  currency: {
    type: String,
    default: 'USD',
  },
});

const chartCanvas = ref(null);
let chartInstance = null;

onMounted(() => {
  createChart();
});

watch(
  () => props.projectionData,
  () => {
    updateChart();
  },
  { deep: true },
);

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.destroy();
  }
});

function createChart() {
  if (!chartCanvas.value) return;

  const ctx = chartCanvas.value.getContext('2d');

  const labels =
    props.projectionData.projectionPoints?.map((point) =>
      new Date(point.date).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
      }),
    ) || [];

  const balances =
    props.projectionData.projectionPoints?.map((point) => point.balance) || [];
  const income =
    props.projectionData.projectionPoints?.map((point) => point.income) || [];
  const expenses =
    props.projectionData.projectionPoints?.map((point) => point.expenses) || [];

  chartInstance = new Chart(ctx, {
    type: 'line',
    data: {
      labels,
      datasets: [
        {
          label: 'Balance',
          data: balances,
          borderColor: '#0d6efd',
          backgroundColor: 'rgba(13, 110, 253, 0.1)',
          fill: true,
          tension: 0.4,
          pointRadius: 4,
          pointHoverRadius: 6,
        },
        {
          label: 'Income',
          data: income,
          borderColor: '#198754',
          backgroundColor: 'rgba(25, 135, 84, 0.1)',
          fill: false,
          tension: 0.4,
          pointRadius: 3,
          borderDash: [5, 5],
        },
        {
          label: 'Expenses',
          data: expenses,
          borderColor: '#dc3545',
          backgroundColor: 'rgba(220, 53, 69, 0.1)',
          fill: false,
          tension: 0.4,
          pointRadius: 3,
          borderDash: [5, 5],
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: {
        mode: 'index',
        intersect: false,
      },
      plugins: {
        legend: {
          display: true,
          position: 'top',
        },
        tooltip: {
          callbacks: {
            label: function (context) {
              let label = context.dataset.label || '';
              if (label) {
                label += ': ';
              }
              if (context.parsed.y !== null) {
                label += new Intl.NumberFormat('en-US', {
                  style: 'currency',
                  currency: props.currency,
                }).format(context.parsed.y);
              }
              return label;
            },
          },
        },
      },
      scales: {
        y: {
          beginAtZero: false,
          ticks: {
            callback: function (value) {
              return new Intl.NumberFormat('en-US', {
                style: 'currency',
                currency: props.currency,
                minimumFractionDigits: 0,
                maximumFractionDigits: 0,
              }).format(value);
            },
          },
        },
      },
    },
  });
}

function updateChart() {
  if (!chartInstance) {
    createChart();
    return;
  }

  const labels =
    props.projectionData.projectionPoints?.map((point) =>
      new Date(point.date).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
      }),
    ) || [];

  const balances =
    props.projectionData.projectionPoints?.map((point) => point.balance) || [];
  const income =
    props.projectionData.projectionPoints?.map((point) => point.income) || [];
  const expenses =
    props.projectionData.projectionPoints?.map((point) => point.expenses) || [];

  chartInstance.data.labels = labels;
  chartInstance.data.datasets[0].data = balances;
  chartInstance.data.datasets[1].data = income;
  chartInstance.data.datasets[2].data = expenses;

  chartInstance.update();
}
</script>

<style scoped>
.balance-chart-container {
  position: relative;
  height: 300px;
  width: 100%;
}
</style>
