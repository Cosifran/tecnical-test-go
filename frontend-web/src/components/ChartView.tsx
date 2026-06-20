// Dual-axis line chart for fuel/temperature — uses CSS variables for Recharts (no className support)
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import type { SensorHistoryPoint } from "@/types/domain";

interface ChartViewProps {
  history: SensorHistoryPoint[] | undefined;
}

const CHART_COLORS = {
  fuel: "var(--color-primary)",
  temperature: "var(--color-warning)",
  text: "var(--color-text)",
  gridLine: "var(--color-border)",
};

export function ChartView({ history }: ChartViewProps) {
  if (!history || history.length === 0) {
    return (
      <div className="flex h-64 items-center justify-center rounded-lg border border-slate-700 bg-slate-800">
        <p className="text-sm text-slate-400">No history data available</p>
      </div>
    );
  }

  const chartData = history.map((point) => ({
    time: new Date(point.timestamp).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    }),
    fuel_level: point.fuel_level,
    temperature: point.temperature,
  }));

  return (
    <div className="rounded-lg border border-slate-700 bg-slate-800 p-4">
      <h3 className="mb-3 text-sm font-semibold text-white">Sensor History</h3>
      <ResponsiveContainer width="100%" height={250}>
        <LineChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.gridLine} />
          <XAxis
            dataKey="time"
            tick={{ fill: CHART_COLORS.text, fontSize: 11 }}
            stroke={CHART_COLORS.gridLine}
          />
          <YAxis
            yAxisId="fuel"
            orientation="left"
            tick={{ fill: CHART_COLORS.text, fontSize: 11 }}
            stroke={CHART_COLORS.gridLine}
            unit="%"
          />
          <YAxis
            yAxisId="temp"
            orientation="right"
            tick={{ fill: CHART_COLORS.text, fontSize: 11 }}
            stroke={CHART_COLORS.gridLine}
            unit="°C"
          />
          <Tooltip
            contentStyle={{
              backgroundColor: "var(--color-surface)",
              border: "1px solid var(--color-border)",
              borderRadius: "8px",
            }}
          />
          <Legend />
          <Line
            yAxisId="fuel"
            type="monotone"
            dataKey="fuel_level"
            name="Fuel %"
            stroke={CHART_COLORS.fuel}
            strokeWidth={2}
            dot={false}
          />
          <Line
            yAxisId="temp"
            type="monotone"
            dataKey="temperature"
            name="Temp °C"
            stroke={CHART_COLORS.temperature}
            strokeWidth={2}
            dot={false}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}