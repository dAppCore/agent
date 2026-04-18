var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
import { LitElement, html, css } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';
/**
 * Agent dashboard panel — shows issues, sprint progress, and fleet status.
 * Works in core/ide (Wails), lthn.sh (Laravel), and standalone browsers.
 *
 * @element core-agent-panel
 */
let CoreAgentPanel = class CoreAgentPanel extends LitElement {
    constructor() {
        super(...arguments);
        this.apiUrl = '';
        this.apiKey = '';
        this.issues = [];
        this.sprint = null;
        this.loading = true;
        this.error = '';
        this.activeTab = 'issues';
    }
    static { this.styles = css `
    :host {
      display: block;
      font-family: 'Inter', system-ui, -apple-system, sans-serif;
      color: #e2e8f0;
      background: #0f172a;
      border-radius: 0.75rem;
      overflow: hidden;
    }

    .header {
      display: flex;
      align-items: centre;
      justify-content: space-between;
      padding: 1rem 1.25rem;
      background: #1e293b;
      border-bottom: 1px solid #334155;
    }

    .header h2 {
      margin: 0;
      font-size: 1rem;
      font-weight: 600;
      color: #f1f5f9;
    }

    .tabs {
      display: flex;
      gap: 0.25rem;
      background: #0f172a;
      border-radius: 0.375rem;
      padding: 0.125rem;
    }

    .tab {
      padding: 0.375rem 0.75rem;
      font-size: 0.75rem;
      font-weight: 500;
      border: none;
      background: transparent;
      color: #94a3b8;
      border-radius: 0.25rem;
      cursor: pointer;
      transition: all 0.15s;
    }

    .tab.active {
      background: #334155;
      color: #f1f5f9;
    }

    .tab:hover:not(.active) {
      color: #cbd5e1;
    }

    .content {
      padding: 1rem 1.25rem;
      max-height: 400px;
      overflow-y: auto;
    }

    .issue-row {
      display: flex;
      align-items: centre;
      justify-content: space-between;
      padding: 0.625rem 0;
      border-bottom: 1px solid #1e293b;
    }

    .issue-row:last-child {
      border-bottom: none;
    }

    .issue-title {
      font-size: 0.875rem;
      color: #e2e8f0;
      flex: 1;
      margin-right: 0.75rem;
    }

    .badge {
      display: inline-block;
      padding: 0.125rem 0.5rem;
      border-radius: 9999px;
      font-size: 0.625rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.025em;
    }

    .badge-open { background: #1e3a5f; color: #60a5fa; }
    .badge-assigned { background: #3b2f63; color: #a78bfa; }
    .badge-in_progress { background: #422006; color: #f59e0b; }
    .badge-review { background: #164e63; color: #22d3ee; }
    .badge-done { background: #14532d; color: #4ade80; }
    .badge-closed { background: #1e293b; color: #64748b; }

    .badge-critical { background: #450a0a; color: #ef4444; }
    .badge-high { background: #431407; color: #f97316; }
    .badge-normal { background: #1e293b; color: #94a3b8; }
    .badge-low { background: #1e293b; color: #64748b; }

    .sprint-card {
      background: #1e293b;
      border-radius: 0.5rem;
      padding: 1.25rem;
    }

    .sprint-title {
      font-size: 1rem;
      font-weight: 600;
      margin-bottom: 0.75rem;
    }

    .progress-bar {
      height: 0.5rem;
      background: #334155;
      border-radius: 9999px;
      overflow: hidden;
      margin-bottom: 0.5rem;
    }

    .progress-fill {
      height: 100%;
      background: linear-gradient(90deg, #8b5cf6, #6366f1);
      border-radius: 9999px;
      transition: width 0.3s ease;
    }

    .progress-stats {
      display: flex;
      gap: 1rem;
      font-size: 0.75rem;
      color: #94a3b8;
    }

    .stat {
      display: flex;
      align-items: centre;
      gap: 0.25rem;
    }

    .stat-value {
      font-weight: 600;
      color: #e2e8f0;
    }

    .empty {
      text-align: centre;
      padding: 2rem;
      color: #64748b;
      font-size: 0.875rem;
    }

    .error {
      text-align: centre;
      padding: 1rem;
      color: #ef4444;
      font-size: 0.875rem;
    }

    .loading {
      text-align: centre;
      padding: 2rem;
      color: #64748b;
    }
  `; }
    connectedCallback() {
        super.connectedCallback();
        this.fetchData();
        // Refresh every 30 seconds
        setInterval(() => this.fetchData(), 30000);
    }
    async fetchData() {
        const base = this.apiUrl || window.location.origin;
        const headers = {
            'Accept': 'application/json',
        };
        if (this.apiKey) {
            headers['Authorization'] = `Bearer ${this.apiKey}`;
        }
        try {
            const [issuesRes, sprintsRes] = await Promise.all([
                fetch(`${base}/v1/issues`, { headers }),
                fetch(`${base}/v1/sprints`, { headers }),
            ]);
            if (issuesRes.ok) {
                const issuesData = await issuesRes.json();
                this.issues = issuesData.data || [];
            }
            if (sprintsRes.ok) {
                const sprintsData = await sprintsRes.json();
                const sprints = sprintsData.data || [];
                this.sprint = sprints.find((s) => s.status === 'active') || sprints[0] || null;
            }
            this.loading = false;
            this.error = '';
        }
        catch (e) {
            this.error = 'Failed to connect to API';
            this.loading = false;
        }
    }
    setTab(tab) {
        this.activeTab = tab;
    }
    renderIssues() {
        if (this.issues.length === 0) {
            return html `<div class="empty">No issues found</div>`;
        }
        return this.issues.map(issue => html `
      <div class="issue-row">
        <span class="issue-title">${issue.title}</span>
        <span class="badge badge-${issue.priority}">${issue.priority}</span>
        <span class="badge badge-${issue.status}" style="margin-left: 0.25rem">${issue.status}</span>
      </div>
    `);
    }
    renderSprint() {
        if (!this.sprint) {
            return html `<div class="empty">No active sprint</div>`;
        }
        const progress = this.sprint.progress;
        return html `
      <div class="sprint-card">
        <div class="sprint-title">${this.sprint.title}</div>
        <span class="badge badge-${this.sprint.status}">${this.sprint.status}</span>
        <div class="progress-bar" style="margin-top: 1rem">
          <div class="progress-fill" style="width: ${progress.percentage}%"></div>
        </div>
        <div class="progress-stats">
          <div class="stat">
            <span class="stat-value">${progress.total}</span> total
          </div>
          <div class="stat">
            <span class="stat-value">${progress.open}</span> open
          </div>
          <div class="stat">
            <span class="stat-value">${progress.in_progress}</span> in progress
          </div>
          <div class="stat">
            <span class="stat-value">${progress.closed}</span> done
          </div>
        </div>
      </div>
    `;
    }
    render() {
        if (this.loading) {
            return html `<div class="loading">Loading...</div>`;
        }
        if (this.error) {
            return html `<div class="error">${this.error}</div>`;
        }
        return html `
      <div class="header">
        <h2>Agent Dashboard</h2>
        <div class="tabs">
          <button class="tab ${this.activeTab === 'issues' ? 'active' : ''}"
                  @click=${() => this.setTab('issues')}>
            Issues (${this.issues.length})
          </button>
          <button class="tab ${this.activeTab === 'sprint' ? 'active' : ''}"
                  @click=${() => this.setTab('sprint')}>
            Sprint
          </button>
        </div>
      </div>
      <div class="content">
        ${this.activeTab === 'issues' ? this.renderIssues() : this.renderSprint()}
      </div>
    `;
    }
};
__decorate([
    property({ type: String, attribute: 'api-url' })
], CoreAgentPanel.prototype, "apiUrl", void 0);
__decorate([
    property({ type: String, attribute: 'api-key' })
], CoreAgentPanel.prototype, "apiKey", void 0);
__decorate([
    state()
], CoreAgentPanel.prototype, "issues", void 0);
__decorate([
    state()
], CoreAgentPanel.prototype, "sprint", void 0);
__decorate([
    state()
], CoreAgentPanel.prototype, "loading", void 0);
__decorate([
    state()
], CoreAgentPanel.prototype, "error", void 0);
__decorate([
    state()
], CoreAgentPanel.prototype, "activeTab", void 0);
CoreAgentPanel = __decorate([
    customElement('core-agent-panel')
], CoreAgentPanel);
export { CoreAgentPanel };
