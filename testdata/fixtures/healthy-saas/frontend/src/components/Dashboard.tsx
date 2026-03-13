// Dashboard component
// Known complexity: renderDashboard=4, formatMetric=3

import React from "react";

interface Metric {
  label: string;
  value: number;
  trend: "up" | "down" | "flat";
}

interface DashboardProps {
  metrics: Metric[];
  isLoading: boolean;
  error?: string;
}

export function Dashboard({ metrics, isLoading, error }: DashboardProps) {
  // complexity: 1 (base) + 3 decision points = 4
  if (isLoading) {
    return <div className="loading">Loading...</div>;
  }

  if (error) {
    return <div className="error">{error}</div>;
  }

  if (metrics.length === 0) {
    return <div className="empty">No metrics available</div>;
  }

  return (
    <div className="dashboard">
      {metrics.map((m) => (
        <div key={m.label} className="metric-card">
          <h3>{m.label}</h3>
          <span>{formatMetric(m)}</span>
        </div>
      ))}
    </div>
  );
}

function formatMetric(metric: Metric): string {
  // complexity: 1 (base) + 2 ternary operators = 3
  const arrow = metric.trend === "up" ? "^" : metric.trend === "down" ? "v" : "-";
  return `${metric.value} ${arrow}`;
}

export default Dashboard;
