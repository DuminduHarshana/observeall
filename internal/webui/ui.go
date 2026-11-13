package webui

// IndexHTML is the single-page application for the ObserveAll visual pipeline router.
const IndexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>ObserveAll — Pipeline Control Plane</title>
  <style>
    :root {
      --bg: #0d1117;
      --card-bg: #161b22;
      --border: #30363d;
      --accent: #58a6ff;
      --green: #2ea043;
      --red: #f85149;
      --text: #c9d1d9;
      --text-bright: #f0f6fc;
      --font: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background-color: var(--bg);
      color: var(--text);
      font-family: var(--font);
      padding: 24px;
      line-height: 1.5;
    }
    header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 24px;
      padding-bottom: 16px;
      border-bottom: 1px solid var(--border);
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    .brand h1 {
      font-size: 22px;
      color: var(--text-bright);
      font-weight: 600;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 10px;
      border-radius: 12px;
      font-size: 12px;
      font-weight: 600;
      background: #21262d;
      border: 1px solid var(--border);
    }
    .badge.up { color: #56d364; border-color: rgba(46,160,67,0.4); }
    .badge.down { color: #f85149; border-color: rgba(248,81,73,0.4); }
    .dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: currentColor;
    }
    .grid {
      display: grid;
      grid-template-columns: 2fr 1fr;
      gap: 20px;
    }
    .card {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 20px;
      margin-bottom: 20px;
    }
    .card h2 {
      font-size: 16px;
      font-weight: 600;
      color: var(--text-bright);
      margin-bottom: 16px;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .pipeline-row {
      display: grid;
      grid-template-columns: 120px 30px 140px 30px 1fr;
      align-items: center;
      gap: 8px;
      padding: 12px 0;
      border-bottom: 1px solid var(--border);
    }
    .pipeline-row:last-child { border-bottom: none; }
    .node {
      background: #21262d;
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 8px 12px;
      text-align: center;
      font-size: 13px;
      font-weight: 600;
      color: var(--text-bright);
    }
    .node.signal {
      background: #1f2937;
      border-color: #38bdf8;
      color: #38bdf8;
    }
    .node.processor {
      background: #2d2036;
      border-color: #d946ef;
      color: #f0abfc;
      font-size: 11px;
    }
    .arrow {
      text-align: center;
      color: #6e7681;
      font-size: 16px;
    }
    .backends-cell {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }
    .chip {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 10px;
      border-radius: 6px;
      font-size: 12px;
      background: #21262d;
      border: 1px solid var(--border);
      cursor: pointer;
      user-select: none;
      transition: all 0.15s ease;
    }
    .chip input { cursor: pointer; }
    .chip.active {
      background: #1c324c;
      border-color: var(--accent);
      color: var(--accent);
    }
    .btn {
      background: var(--green);
      color: #fff;
      border: none;
      border-radius: 6px;
      padding: 10px 18px;
      font-size: 14px;
      font-weight: 600;
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      gap: 8px;
    }
    .btn:hover { background: #2c973b; }
    .stat-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    .stat-item {
      display: flex;
      justify-content: space-between;
      padding: 8px 12px;
      background: #21262d;
      border-radius: 6px;
      font-size: 13px;
    }
    .stat-item span:first-child { color: #8b949e; }
    .stat-item span:last-child { font-weight: 600; color: var(--text-bright); font-family: monospace; }
    .alert {
      padding: 10px 14px;
      border-radius: 6px;
      margin-top: 14px;
      font-size: 13px;
      display: none;
    }
    .alert.success { background: rgba(46,160,67,0.15); border: 1px solid var(--green); color: #56d364; }
    .alert.error { background: rgba(248,81,73,0.15); border: 1px solid var(--red); color: #f85149; }
  </style>
</head>
<body>
  <header>
    <div class="brand">
      <span style="font-size:24px;">🔭</span>
      <div>
        <h1>ObserveAll Pipeline Control Plane</h1>
        <p style="font-size:12px; color:#8b949e;">Visual Signal Routing &amp; Live Ingestion Monitor</p>
      </div>
    </div>
    <div id="collectorStatus" class="badge">
      <span class="dot"></span>
      <span id="statusText">Checking collector...</span>
    </div>
  </header>

  <div class="grid">
    <div>
      <div class="card">
        <h2>🔀 Signal Routing Matrix</h2>
        <p style="font-size:13px; color:#8b949e; margin-bottom:16px;">
          Select target backends for each telemetry signal. The collector pipeline YAML updates and applies instantly.
        </p>

        <div id="routingTable">
          <!-- Populated by JS -->
        </div>

        <div style="margin-top: 20px; display:flex; justify-content:space-between; align-items:center;">
          <button id="saveBtn" class="btn" onclick="saveRoutes()">
            <span>💾</span> Save &amp; Apply Pipeline
          </button>
          <div id="saveFeedback" class="alert"></div>
        </div>
      </div>
    </div>

    <div>
      <div class="card">
        <h2>📊 Live Collector Stats</h2>
        <div class="stat-list">
          <div class="stat-item">
            <span>Project</span>
            <span id="statProject">-</span>
          </div>
          <div class="stat-item">
            <span>Environment</span>
            <span id="statEnv">-</span>
          </div>
          <div class="stat-item">
            <span>Accepted Spans</span>
            <span id="statSpans">0</span>
          </div>
          <div class="stat-item">
            <span>Accepted Metrics</span>
            <span id="statMetrics">0</span>
          </div>
          <div class="stat-item">
            <span>Tail-Sampling</span>
            <span id="statSampling">-</span>
          </div>
          <div class="stat-item">
            <span>PII Scrubbing</span>
            <span id="statPII">-</span>
          </div>
        </div>
      </div>
    </div>
  </div>

  <script>
    const ALL_BACKENDS = ["prometheus", "grafana", "datadog", "dynatrace", "newrelic", "jaeger", "custom-otlp"];
    const SIGNALS = ["traces", "metrics", "logs"];
    let currentConfig = null;

    async function loadPipeline() {
      try {
        const res = await fetch("/api/pipeline");
        currentConfig = await res.json();
        renderMatrix();
        updateProjectStats();
      } catch (err) {
        console.error("Failed to load pipeline:", err);
      }
    }

    function renderMatrix() {
      const container = document.getElementById("routingTable");
      container.innerHTML = "";

      SIGNALS.forEach(sig => {
        const row = document.createElement("div");
        row.className = "pipeline-row";

        const sigNode = document.createElement("div");
        sigNode.className = "node signal";
        sigNode.textContent = sig.toUpperCase();

        const arrow1 = document.createElement("div");
        arrow1.className = "arrow";
        arrow1.textContent = "➔";

        const procNode = document.createElement("div");
        procNode.className = "node processor";
        procNode.textContent = "Batch + Filter";

        const arrow2 = document.createElement("div");
        arrow2.className = "arrow";
        arrow2.textContent = "➔";

        const backendsCell = document.createElement("div");
        backendsCell.className = "backends-cell";

        ALL_BACKENDS.forEach(b => {
          const isChecked = (currentConfig.backends || []).includes(b);
          const label = document.createElement("label");
          label.className = "chip" + (isChecked ? " active" : "");

          const input = document.createElement("input");
          input.type = "checkbox";
          input.value = b;
          input.dataset.signal = sig;
          input.checked = isChecked;
          input.onchange = () => {
            label.classList.toggle("active", input.checked);
          };

          label.appendChild(input);
          label.appendChild(document.createTextNode(b));
          backendsCell.appendChild(label);
        });

        row.appendChild(sigNode);
        row.appendChild(arrow1);
        row.appendChild(procNode);
        row.appendChild(arrow2);
        row.appendChild(backendsCell);
        container.appendChild(row);
      });
    }

    async function pollStatus() {
      try {
        const res = await fetch("/api/status");
        const status = await res.json();
        const badge = document.getElementById("collectorStatus");
        const txt = document.getElementById("statusText");

        if (status.healthy) {
          badge.className = "badge up";
          txt.textContent = "Collector: Healthy (8888)";
        } else {
          badge.className = "badge down";
          txt.textContent = "Collector: Unreachable";
        }

        document.getElementById("statSpans").textContent = (status.accepted_spans || 0).toLocaleString();
        document.getElementById("statMetrics").textContent = (status.accepted_metric_points || 0).toLocaleString();
      } catch (err) {
        const badge = document.getElementById("collectorStatus");
        badge.className = "badge down";
        document.getElementById("statusText").textContent = "Collector: Offline";
      }
    }

    function updateProjectStats() {
      if (!currentConfig) return;
      document.getElementById("statProject").textContent = currentConfig.name || "observeall";
      document.getElementById("statEnv").textContent = currentConfig.environment || "dev";
      document.getElementById("statSampling").textContent = currentConfig.sampling || "tail";
      document.getElementById("statPII").textContent = currentConfig.pii_scrubbing ? "Enabled" : "Disabled";
    }

    async function saveRoutes() {
      const selected = new Set();
      document.querySelectorAll('#routingTable input[type="checkbox"]:checked').forEach(cb => {
        selected.add(cb.value);
      });

      const backends = Array.from(selected);
      const feedback = document.getElementById("saveFeedback");

      try {
        const res = await fetch("/api/pipeline/route", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ backends: backends })
        });
        const data = await res.json();

        if (res.ok) {
          feedback.className = "alert success";
          feedback.textContent = "✔ Pipeline updated & collector-config.yaml regenerated!";
          feedback.style.display = "block";
          loadPipeline();
        } else {
          feedback.className = "alert error";
          feedback.textContent = "✖ Error: " + (data.error || "Save failed");
          feedback.style.display = "block";
        }
      } catch (err) {
        feedback.className = "alert error";
        feedback.textContent = "✖ Connection failed: " + err.message;
        feedback.style.display = "block";
      }
      setTimeout(() => { feedback.style.display = "none"; }, 4000);
    }

    loadPipeline();
    pollStatus();
    setInterval(pollStatus, 5000);
  </script>
</body>
</html>`

