import { useEffect, useState } from "react";

export interface LiveStatsData {
  total_requests: number;
  active_requests: number;
  cache_hits: number;
  failed_requests: number;
  tester_speed: number;
  tester_pending: number;
}

const INITIAL_STATS: LiveStatsData = {
  total_requests: 0,
  active_requests: 0,
  cache_hits: 0,
  failed_requests: 0,
  tester_speed: 0,
  tester_pending: 0,
};

const getSseUrl = () => {
  const apiBase = import.meta.env.VITE_API_BASE_URL || "";
  const normalizedBase =
    apiBase && apiBase !== "/" && !apiBase.startsWith("/")
      ? apiBase.replace(/\/$/, "")
      : "";

  return `${normalizedBase}/api/v2/stats/live`;
};

export const useLiveStats = () => {
  const [stats, setStats] = useState<LiveStatsData>(INITIAL_STATS);
  const [history, setHistory] = useState<{ x: number; y: number }[]>([]);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const evtSource = new EventSource(getSseUrl());

    evtSource.onopen = () => {
      setConnected(true);
    };

    evtSource.onmessage = (event) => {
      const data = JSON.parse(event.data) as LiveStatsData;
      setStats(data);
      setConnected(true);

      setHistory((prev) => {
        const next = [
          ...prev,
          { x: new Date().getTime(), y: data.active_requests },
        ];

        return next.slice(-20);
      });
    };

    evtSource.onerror = () => {
      setConnected(false);
    };

    return () => {
      evtSource.close();
    };
  }, []);

  return { stats, history, connected };
};