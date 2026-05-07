<script setup>
import { computed, onBeforeMount, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { DateTime } from 'luxon';
import { Bar } from 'vue-chartjs';
import {
  BarElement,
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LinearScale,
  Title,
  Tooltip,
} from 'chart.js';

import { Services } from '@/services/servicesConfig';
import { processError } from '@/errors/errorHandlers';
import ReportDateFilters from '@/components/reports/ReportDateFilters.vue';

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  BarElement,
  CategoryScale,
  LinearScale,
);

const router = useRouter();
const loaded = ref(false);

const chartData = reactive({
  labels: [],
  datasets: [
    {
      label: 'Total Income',
      backgroundColor: 'rgba(72,234,12,0.83)',
      data: [],
    },
    {
      label: 'Total Expenses',
      backgroundColor: 'rgba(245,28,70,0.83)',
      data: [],
    },
    {
      label: 'Net Flow',
      backgroundColor: 'rgba(12,53,199,0.82)',
      data: [],
    },
  ],
});
const cashFlowReport = ref({});

const chartOptions = reactive({
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    y: {
      beginAtZero: true,
    },
  },
});

// Computed totals for the summary row
const totals = computed(() => {
  let totalIncome = 0;
  let totalExpenses = 0;
  let totalNetFlow = 0;

  if (cashFlowReport.value.totalIncome) {
    for (let period in cashFlowReport.value.totalIncome) {
      totalIncome += cashFlowReport.value.totalIncome[period] || 0;
      totalExpenses += cashFlowReport.value.totalExpenses[period] || 0;
      totalNetFlow += cashFlowReport.value.netFlow[period] || 0;
    }
  }

  return {
    income: totalIncome,
    expenses: totalExpenses,
    netFlow: totalNetFlow,
  };
});

const startDate = ref('');
const endDate = ref(DateTime.now().toISODate());
const period = ref('monthly');

onBeforeMount(async () => {
  try {
    await getReportData();
  } catch (e) {
    await processError(e, router);
  }
});

async function updateChartData() {
  await getReportData();
}

async function getReportData() {
  clearChartData();

  const filters = {
    period: period.value,
  };
  if (startDate.value !== '') {
    filters.startDate = startDate.value;
  }
  if (endDate.value !== '') {
    filters.endDate = endDate.value;
  }
  cashFlowReport.value = await Services.reportsService.getReport(
    'cashflow',
    filters,
  );

  for (let period in cashFlowReport.value.netFlow) {
    chartData.labels.push(period);
    chartData.datasets[0].data.push(cashFlowReport.value.totalIncome[period]);
    chartData.datasets[1].data.push(
      -1 * cashFlowReport.value.totalExpenses[period],
    );
    chartData.datasets[2].data.push(cashFlowReport.value.netFlow[period]);
  }

  loaded.value = true;
}

function clearChartData() {
  chartData.labels = [];
  chartData.datasets[0].data = [];
  chartData.datasets[1].data = [];
  chartData.datasets[2].data = [];
}

async function onFiltersChange() {
  loaded.value = false;
  await updateChartData();
  loaded.value = true;
}
</script>

<template>
  <div class="section-card">
    <h2>{{ $t('message.cashFlowReport') }}</h2>

    <ReportDateFilters
      v-model:start-date="startDate"
      v-model:end-date="endDate"
      v-model:period="period"
      show-period
      @change="onFiltersChange" />

    <div style="height: 340px; margin-top: 20px">
      <Bar
        v-if="loaded"
        :data="chartData"
        :options="chartOptions" />
    </div>
  </div>

  <div
    class="section-card"
    style="margin-top: 16px">
    <div class="table-header-grid">
      <div class="left">{{ $t('message.period') }}</div>
      <div class="right">{{ $t('message.totalIncome') }}</div>
      <div class="right">{{ $t('message.totalExpenses') }}</div>
      <div class="right">{{ $t('message.netFlow') }}</div>
    </div>

    <div
      v-for="(item, index) in cashFlowReport.netFlow"
      :key="index"
      class="table-row-grid">
      <div class="cell">
        <span class="th">{{ $t('message.period') }}</span>
        <span class="val">{{ index }}</span>
      </div>
      <div class="cell right">
        <span class="th">{{ $t('message.totalIncome') }}</span>
        <span class="val">
          {{ $n(cashFlowReport.totalIncome[index] ?? 0, 'decimal') }}
          {{ cashFlowReport.currency }}
        </span>
      </div>
      <div class="cell right">
        <span class="th">{{ $t('message.totalExpenses') }}</span>
        <span class="val">
          {{ $n(cashFlowReport.totalExpenses[index] ?? 0, 'decimal') }}
          {{ cashFlowReport.currency }}
        </span>
      </div>
      <div class="cell right">
        <span class="th">{{ $t('message.netFlow') }}</span>
        <span class="val">
          {{ $n(cashFlowReport.netFlow[index] ?? 0, 'decimal') }}
          {{ cashFlowReport.currency }}
        </span>
      </div>
    </div>

    <!-- Totals row -->
    <div class="table-row-grid totals-row">
      <div class="cell">
        <span class="th">{{ $t('message.total') }}</span>
        <span class="val">{{ $t('message.total') }}</span>
      </div>
      <div class="cell right">
        <span class="th">{{ $t('message.totalIncome') }}</span>
        <span class="val">
          {{ $n(totals.income, 'decimal') }} {{ cashFlowReport.currency }}
        </span>
      </div>
      <div class="cell right">
        <span class="th">{{ $t('message.totalExpenses') }}</span>
        <span class="val">
          {{ $n(totals.expenses, 'decimal') }} {{ cashFlowReport.currency }}
        </span>
      </div>
      <div class="cell right">
        <span class="th">{{ $t('message.netFlow') }}</span>
        <span class="val">
          {{ $n(totals.netFlow, 'decimal') }} {{ cashFlowReport.currency }}
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.table-header-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr 1fr;
  font-weight: 600;
  border-bottom: 2px solid #000;
  padding: 8px 0;
  margin-bottom: 6px;
}
.table-header-grid .left {
  text-align: left;
}
.table-header-grid .right {
  text-align: right;
}

.table-row-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr 1fr;
  align-items: center;
  padding: 8px 0;
  transition: background 0.15s;
}
.table-row-grid:hover {
  background: #f0f0f0;
}

.totals-row {
  font-weight: 700;
  border-top: 2px solid #000;
  margin-top: 4px;
  padding-top: 12px;
}
.totals-row:hover {
  background: transparent;
}

.cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
.cell.right {
  justify-content: flex-end;
}

.th {
  display: none;
  color: #6c757d;
}

@media (max-width: 576px) {
  .table-header-grid {
    display: none;
  }

  .table-row-grid {
    grid-template-columns: 1fr;
    row-gap: 6px;
    padding: 10px 0;
  }

  .cell {
    justify-content: space-between;
  }
  .cell.right {
    justify-content: space-between;
  }

  .th {
    display: inline;
    font-weight: 600;
  }
  .val {
    text-align: right;
  }
}

@media (max-width: 767px) {
  .section-card {
    padding-left: 5px;
    padding-right: 5px;
  }
}
</style>
